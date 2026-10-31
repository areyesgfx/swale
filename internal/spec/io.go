package spec

import (
	"bytes"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// FileName is the name of the spec file at the root of every swale project.
const FileName = "swale.yaml"

// Comments Save writes into swale.yaml, for whoever opens it next. Load
// ignores them.
const (
	fileComment = `swale.yaml describes your app. After editing it, run "swale generate" to
create any files that are missing. swale never overwrites a file that
already exists, so to apply a change to one, delete it first.`

	stateBucketComment = `The S3 bucket that stores Terraform state. It was picked when the project
was created; the random suffix keeps it from colliding with anyone else's,
since bucket names are global. Don't change it once you've deployed.`
)

// Load reads and decodes a swale.yaml file from path, then validates it.
// Unknown fields are rejected so a typo in a key surfaces as an error
// rather than being silently dropped.
func Load(path string) (Spec, error) {
	f, err := os.Open(path)
	if err != nil {
		return Spec{}, fmt.Errorf("open spec %q: %w", path, err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var s Spec
	if err := dec.Decode(&s); err != nil {
		return Spec{}, fmt.Errorf("decode spec %q: %w", path, err)
	}
	if err := s.Validate(); err != nil {
		return Spec{}, fmt.Errorf("invalid spec %q: %w", path, err)
	}
	return s, nil
}

// Save validates spec and writes it to path as commented YAML. An invalid
// spec is never written, so any swale.yaml on disk is one swale can load
// back.
func Save(spec Spec, path string) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("refusing to save invalid spec: %w", err)
	}

	// Encoding into a yaml.Node first, instead of marshaling straight to
	// bytes, is what makes room for comments.
	var doc yaml.Node
	if err := doc.Encode(spec); err != nil {
		return fmt.Errorf("encode spec: %w", err)
	}
	doc.HeadComment = fileComment
	for i := 0; i < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "state_bucket" {
			doc.Content[i].HeadComment = stateBucketComment
		}
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("encode spec: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode spec: %w", err)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write spec %q: %w", path, err)
	}
	return nil
}
