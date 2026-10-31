# Each unit's .terraform.lock.hcl pins the exact provider version; this
# bound keeps an upgrade of that lock file within the same major version.
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}
