include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${get_terragrunt_dir()}/../../../../../modules/networking"
}

inputs = {
  vpc_cidr = values.vpc_cidr
  nat_mode = values.nat_mode
}
