package spec

// AccountMode selects single-account vs multi-account topology.
type AccountMode string

const (
	AccountModeSingle AccountMode = "single"
	AccountModeMulti  AccountMode = "multi"
)

// AccountsSpec describes the AWS account topology. In AccountModeSingle,
// Accounts holds exactly one entry. In AccountModeMulti, it holds one
// workload account per environment plus one management account.
type AccountsSpec struct {
	Mode     AccountMode `yaml:"mode"`
	Accounts []Account   `yaml:"accounts"`
}

// Account is an AWS account folder under live/<account>/.
type Account struct {
	Name         string `yaml:"name"`
	IsManagement bool   `yaml:"is_management"`
}
