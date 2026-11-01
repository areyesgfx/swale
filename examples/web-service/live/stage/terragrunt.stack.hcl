unit "networking" {
  source = "../../units/networking"
  path   = "networking"
  values = {
    name     = "acme-api-stage"
    vpc_cidr = "10.20.0.0/16"
    nat_mode = "single"
  }
}

unit "database" {
  source = "../../units/database"
  path   = "database"
  values = {
    name                  = "acme-api-stage"
    instance_class        = "db.t4g.micro"
    highly_available      = false
    deletion_protection   = false
    backup_retention_days = 7
    password_version      = 1
  }
}

unit "service" {
  source = "../../units/service"
  path   = "service"
  values = {
    name          = "acme-api-stage"
    image_tag     = "v1"
    desired_count = 1

    port              = 8080
    health_check_path = "/health"
  }
}
