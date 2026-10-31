unit "networking" {
  source = "../../units/networking"
  path   = "networking"
  values = {
    name     = "acme-api-prod"
    vpc_cidr = "10.30.0.0/16"
    nat_mode = "per_az"
  }
}
