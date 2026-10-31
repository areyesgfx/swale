unit "networking" {
  source = "../../../units/networking"
  path   = "networking"
  values = {
    vpc_cidr = "10.0.0.0/16"
    nat_mode = "single"
  }
}
