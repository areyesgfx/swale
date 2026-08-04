package spec

// NATMode selects the NAT gateway topology. Defaults are single for dev,
// per_az for prod (enforced by the CLI/wizard, not this type).
type NATMode string

const (
	NATModeSingle NATMode = "single"
	NATModePerAZ  NATMode = "per_az"
)

// NetworkingSpec holds the VPC/subnet/NAT/security-group/endpoint knobs
// for a single environment.
type NetworkingSpec struct {
	VPCCIDR string  `yaml:"vpc_cidr"`
	NAT     NATMode `yaml:"nat"`
}
