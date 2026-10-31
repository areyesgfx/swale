# The provider version is bounded here because generated projects don't
# keep a lock file: Terragrunt writes .terraform.lock.hcl into
# .terragrunt-stack/, which is a gitignored build artifact.
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}
