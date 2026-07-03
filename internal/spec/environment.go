package spec

// Environment is a single deployment environment, e.g. "dev" or "prod".
// Account must match the Name of an entry in AccountsSpec.Accounts.
type Environment struct {
	Name       string
	Account    string
	Networking NetworkingSpec
}
