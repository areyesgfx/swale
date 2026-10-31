unit "networking" {
  source = "../../units/networking"
  path   = "networking"
  values = {
    name     = "acme-jobs-prod"
    vpc_cidr = "10.30.0.0/16"
    nat_mode = "per_az"
  }
}

unit "service" {
  source = "../../units/service"
  path   = "service"
  values = {
    name          = "acme-jobs-prod"
    image_tag     = "v1"
    desired_count = 2
  }
}
