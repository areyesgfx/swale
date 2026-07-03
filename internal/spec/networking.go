package spec

// NATMode selects the NAT gateway topology for a networking stack.
type NATMode string

const (
	// NATModeSingle provisions one NAT gateway shared across all AZs.
	// This is the default for dev environments.
	NATModeSingle NATMode = "single"

	// NATModePerAZ provisions one NAT gateway per availability zone.
	// This is the default for prod environments.
	NATModePerAZ NATMode = "per_az"
)

// NetworkingSpec holds the knobs for an environment's networking category:
// VPC, public/private subnets, NAT, a baseline security group, and S3 VPC
// endpoints.
type NetworkingSpec struct {
	// VPCCIDR is the CIDR block for the VPC, e.g. "10.0.0.0/16".
	VPCCIDR string

	NAT NATMode
}
