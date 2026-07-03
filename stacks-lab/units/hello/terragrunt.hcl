include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${get_terragrunt_dir()}/../../../../modules/hello"
}

inputs = {
  name = values.env
}
