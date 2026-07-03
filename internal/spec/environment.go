package spec

// Environment is a single deployment environment, e.g. "dev" or "prod".
type Environment struct {
	// Name is the environment identifier, used as the live/<account>/<env>/
	// directory name. Must be unique within a Spec.
	Name string

	// Account is the Name of the Account (in AccountsSpec.Accounts) this
	// environment deploys into.
	Account string

	Networking NetworkingSpec
}
