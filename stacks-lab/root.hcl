locals {
  relative_path = path_relative_to_include()
}

generate "backend" {
  path      = "backend.tf"
  if_exists = "overwrite"
  contents  = <<-EOF
    terraform {
      backend "local" {
        path = "${get_repo_root()}/stacks-lab/.tfstate/${local.relative_path}/terraform.tfstate"
      }
    }
  EOF
}

# Written purely so we can inspect what path_relative_to_include() actually
# resolves to once a unit is materialized under .terragrunt-stack/ - see
# stacks-lab/FINDINGS.md.
generate "debug_relative_path" {
  path      = "DEBUG_relative_path.txt"
  if_exists = "overwrite"
  contents  = local.relative_path
}
