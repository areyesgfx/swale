unit "networking" {
  source = "../../units/networking"
  path   = "networking"
  values = {
    name     = "acme-api-prod"
    vpc_cidr = "10.30.0.0/16"
    nat_mode = "per_az"
  }
}

unit "database" {
  source = "../../units/database"
  path   = "database"
  values = {
    name                  = "acme-api-prod"
    instance_class        = "db.t4g.small"
    highly_available      = true
    deletion_protection   = true
    backup_retention_days = 14
    password_version      = 1
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
