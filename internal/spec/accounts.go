package spec

// AccountMode selects whether a Spec targets a single AWS account or a
// multi-account topology with a central management account.
type AccountMode string

const (
	// AccountModeSingle deploys every environment into one AWS account.
	// The generated layout still has exactly one live/<account>/ folder;
	// it's the degenerate case of the multi-account layout.
	AccountModeSingle AccountMode = "single"

	// AccountModeMulti deploys each environment into its own AWS account,
	// plus a central management account for shared state/identity.
	AccountModeMulti AccountMode = "multi"
)

// AccountsSpec describes the AWS account topology.
type AccountsSpec struct {
	Mode AccountMode

	// Accounts are the account folders under live/. In AccountModeSingle
	// this holds exactly one entry with IsManagement false, and every
	// Environment implicitly belongs to it. In AccountModeMulti this
	// holds one workload account per environment plus one management
	// account.
	Accounts []Account
}

// Account is a single AWS account folder under live/<account>/.
type Account struct {
	// Name is the account folder name, e.g. "main" in AccountModeSingle,
	// or "dev"/"prod"/"mgmt" in AccountModeMulti.
	Name string

	// IsManagement marks the central account used for multi-account
	// shared state/identity resources. Always false in AccountModeSingle.
	IsManagement bool
}
