unit "networking" {
  source = "../../../units/networking"
  path   = "networking"
  values = {
    vpc_cidr = "10.1.0.0/16"
    nat_mode = "per_az"
  }
}
