unit "networking" {
  source = "../../units/networking"
  path   = "networking"
  values = {
    name     = "acme-api-prod"
    vpc_cidr = "10.30.0.0/16"
    nat_mode = "per_az"
  }
}

unit "service" {
  source = "../../units/service"
  path   = "service"
  values = {
    name          = "acme-api-prod"
    image_tag     = "v1"
    desired_count = 2

    port              = 8080
    health_check_path = "/health"
  }
}
