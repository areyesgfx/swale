unit "networking" {
  source = "../../units/networking"
  path   = "networking"
  values = {
    name     = "acme-api-dev"
    vpc_cidr = "10.10.0.0/16"
    nat_mode = "single"
  }
}
