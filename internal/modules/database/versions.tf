# Each unit's .terraform.lock.hcl pins the exact provider version; this
# bound keeps an upgrade of that lock file within the same major version.
# Terraform 1.11 is the first with the write-only arguments that keep the
# database password out of state.
terraform {
  required_version = ">= 1.11.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}
