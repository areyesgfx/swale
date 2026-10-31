package spec

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
)

var (
	// regionPattern is a format check (e.g. "us-east-1", "us-gov-west-1"),
	// not an exhaustive allow-list - AWS adds regions over time, so
	// validating against a hardcoded list would rot.
	regionPattern = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d+$`)

	namePattern            = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	bucketPattern          = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	healthCheckPathPattern = regexp.MustCompile(`^/[A-Za-z0-9._~/-]*$`)
)

// maxNameLength keeps "<name>-<environment>" within 32 characters, the
// limit AWS puts on some resource names, such as load balancers.
const maxNameLength = 20

// Validate reports whether a Spec is internally consistent and safe to
// generate from. It collects every problem it finds and returns them
// joined together (errors.Join), so callers see all of them at once rather
// than fixing one, re-running, and hitting the next.
func (s Spec) Validate() error {
	var errs []error

	if err := CheckName(s.Name); err != nil {
		errs = append(errs, fmt.Errorf("name: %w", err))
	}
	if err := CheckRegion(s.Region); err != nil {
		errs = append(errs, fmt.Errorf("region: %w", err))
	}

	if len(s.Environments) == 0 {
		errs = append(errs, errors.New("environments: at least one environment is required"))
	}
	seen := make(map[string]bool, len(s.Environments))
	for i, env := range s.Environments {
		switch {
		case !slices.Contains(KnownEnvironments, env):
			errs = append(errs, fmt.Errorf("environments[%d]: %q is not one of %v", i, env, KnownEnvironments))
		case seen[env]:
			errs = append(errs, fmt.Errorf("environments[%d]: %q is listed more than once", i, env))
		}
		seen[env] = true
	}

	errs = append(errs, s.App.validate()...)

	if err := checkBucket(s.StateBucket); err != nil {
		errs = append(errs, fmt.Errorf("state_bucket: %w", err))
	}

	return errors.Join(errs...)
}

func (a App) validate() []error {
	if !a.Containerized {
		if a.Kind != "" || a.Port != 0 || a.HealthCheckPath != "" {
			return []error{errors.New("app: kind, port and health_check_path only apply when containerized is true")}
		}
		return nil
	}

	var errs []error
	switch a.Kind {
	case AppKindWeb:
		if err := CheckPort(a.Port); err != nil {
			errs = append(errs, fmt.Errorf("app.port: %w", err))
		}
		if err := CheckHealthCheckPath(a.HealthCheckPath); err != nil {
			errs = append(errs, fmt.Errorf("app.health_check_path: %w", err))
		}
	case AppKindWorker:
		if a.Port != 0 || a.HealthCheckPath != "" {
			errs = append(errs, errors.New("app: port and health_check_path only apply to web apps, since nothing can reach a worker"))
		}
	case "":
		errs = append(errs, fmt.Errorf("app.kind: must not be empty for a containerized app (want %q or %q)", AppKindWeb, AppKindWorker))
	default:
		errs = append(errs, fmt.Errorf("app.kind: %q is not a valid kind (want %q or %q)", a.Kind, AppKindWeb, AppKindWorker))
	}
	return errs
}

// CheckPort is the web app port rule Validate applies, exported for the
// same reason as CheckName.
func CheckPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%d is not a port number (want 1 to 65535)", port)
	}
	return nil
}

// CheckHealthCheckPath is the health check path rule Validate applies,
// exported for the same reason as CheckName. It's an allow-list because
// the path is rendered inside an HCL string, where a quote or "${" would
// change what the generated code means.
func CheckHealthCheckPath(path string) error {
	if !healthCheckPathPattern.MatchString(path) {
		return fmt.Errorf("%q must be a path like \"/health\": a \"/\" followed by letters, numbers, and . _ ~ / -", path)
	}
	return nil
}

// CheckName is the app name rule Validate applies, exported so the wizard
// can reject a bad value as it's typed instead of after the whole form.
func CheckName(name string) error {
	switch {
	case name == "":
		return errors.New("must not be empty")
	case len(name) > maxNameLength:
		return fmt.Errorf("%q is longer than %d characters", name, maxNameLength)
	case !namePattern.MatchString(name):
		return fmt.Errorf("%q must be lowercase letters, numbers and single dashes, starting with a letter (e.g. \"acme-api\")", name)
	}
	return nil
}

// CheckRegion is the region rule Validate applies, exported for the same
// reason as CheckName.
func CheckRegion(region string) error {
	if region == "" {
		return errors.New("must not be empty")
	}
	if !regionPattern.MatchString(region) {
		return fmt.Errorf("%q does not look like an AWS region (e.g. \"us-east-1\")", region)
	}
	return nil
}

// checkBucket applies the S3 bucket naming rules that a name built by
// NewStateBucket, or edited by hand, could break. Dots are rejected too,
// even though S3 allows them, because they break TLS for virtual-hosted
// bucket URLs.
func checkBucket(bucket string) error {
	switch {
	case bucket == "":
		return errors.New("must not be empty")
	case len(bucket) < 3 || len(bucket) > 63:
		return fmt.Errorf("%q must be 3 to 63 characters", bucket)
	case !bucketPattern.MatchString(bucket):
		return fmt.Errorf("%q must be lowercase letters, numbers and dashes, starting and ending with a letter or number", bucket)
	}
	return nil
}
