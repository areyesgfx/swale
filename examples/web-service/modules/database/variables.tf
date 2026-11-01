variable "name" {
  type        = string
  description = "Name for the database's resources, e.g. \"acme-api-prod\". Also its RDS identifier, and, with dashes turned into underscores, the name of the database the app connects to."
}

variable "vpc_id" {
  type        = string
  description = "VPC to run the database in."
}

variable "private_subnet_ids" {
  type        = list(string)
  description = "Subnets the database may run in, in at least two availability zones (RDS requires two even for a single instance)."
}

variable "instance_class" {
  type        = string
  description = "RDS instance class, e.g. \"db.t4g.micro\"."
}

variable "highly_available" {
  type        = bool
  description = "Run a standby copy in another availability zone that RDS fails over to automatically."
}

variable "deletion_protection" {
  type        = bool
  description = "Refuse to delete the database. Set to false and apply before destroying it."
}

variable "backup_retention_days" {
  type        = number
  description = "How many days of automatic daily backups to keep, from 1 to 35."

  validation {
    condition     = var.backup_retention_days >= 1 && var.backup_retention_days <= 35
    error_message = "backup_retention_days must be between 1 and 35."
  }
}

variable "password_version" {
  type        = number
  description = "Increase it to give the database a new master password: the next apply generates one, stores it in the secret and sets it on the database. Restart the app afterwards so it picks the new one up."
  default     = 1
}
