include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${get_terragrunt_dir()}/../../../../modules/database"
}

# Pass --feature mock_dependencies=true to validate without AWS
# credentials, e.g. in CI: the network's outputs then come from
# mock_outputs instead of its state.
feature "mock_dependencies" {
  default = false
}

dependency "networking" {
  config_path  = "../networking"
  skip_outputs = feature.mock_dependencies.value

  # Placeholders so validate and plan work before the network exists.
  # apply always waits for the network and uses its real outputs.
  mock_outputs = {
    vpc_id             = "vpc-00000000000000000"
    private_subnet_ids = ["subnet-00000000000000000", "subnet-00000000000000001"]
  }
  mock_outputs_allowed_terraform_commands = ["init", "validate", "plan"]
}

inputs = {
  name                  = values.name
  vpc_id                = dependency.networking.outputs.vpc_id
  private_subnet_ids    = dependency.networking.outputs.private_subnet_ids
  instance_class        = values.instance_class
  highly_available      = values.highly_available
  deletion_protection   = values.deletion_protection
  backup_retention_days = values.backup_retention_days
  password_version      = values.password_version
}
