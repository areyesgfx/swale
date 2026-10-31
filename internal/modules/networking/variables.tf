variable "name" {
  type        = string
  description = "Prefix for the network's resource names, shown in the AWS console as their Name tag, e.g. \"acme-api-prod\"."
}

variable "vpc_cidr" {
  type        = string
  description = "CIDR block for the VPC. Subnets are carved out of this in equal /20-or-smaller chunks (16 possible chunks), so a /16 or smaller-prefix (larger) source CIDR gives the most usable subnet sizes."
}

variable "az_count" {
  type        = number
  description = "Number of availability zones to spread public/private subnet pairs across."
  default     = 2

  validation {
    condition     = var.az_count >= 2 && var.az_count <= 8
    error_message = "az_count must be between 2 and 8 (8 az_count * 2 subnets per az = 16, the most /20-or-smaller chunks a /16 vpc_cidr can hold)."
  }
}

variable "nat_mode" {
  type        = string
  description = "NAT gateway topology: \"single\" (one NAT gateway shared by every private subnet) or \"per_az\" (one NAT gateway per availability zone, so an AZ outage can't take down another AZ's egress path)."

  validation {
    condition     = contains(["single", "per_az"], var.nat_mode)
    error_message = "nat_mode must be \"single\" or \"per_az\"."
  }
}
