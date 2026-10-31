unit "networking" {
  source = "../../units/networking"
  path   = "networking"
  values = {
    name     = "acme-api-stage"
    vpc_cidr = "10.20.0.0/16"
    nat_mode = "single"
  }
}
