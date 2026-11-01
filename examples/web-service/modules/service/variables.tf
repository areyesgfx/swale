variable "name" {
  type        = string
  description = "Name for the app's resources in this environment, e.g. \"acme-api-prod\". Also the name of its ECR repository, ECS cluster and service, and load balancer, so it must be 32 characters or fewer."

  validation {
    condition     = length(var.name) <= 32
    error_message = "name must be 32 characters or fewer, the limit for load balancer names."
  }
}

variable "vpc_id" {
  type        = string
  description = "VPC to run the app in."
}

variable "private_subnet_ids" {
  type        = list(string)
  description = "Subnets the app's containers and internal load balancer run in. Spread them across availability zones so a zone outage leaves containers running in the others."
}

variable "port" {
  type        = number
  description = "Port the container listens on for web requests. null runs the app as a background worker with no load balancer."
  default     = null

  validation {
    condition     = var.port == null || (var.port >= 1 && var.port <= 65535)
    error_message = "port must be between 1 and 65535, or null for a background worker."
  }
}

variable "health_check_path" {
  type        = string
  description = "Path the load balancer requests to check that a container is healthy. It must answer with a 2xx or 3xx status."
  default     = "/"

  validation {
    condition     = startswith(var.health_check_path, "/")
    error_message = "health_check_path must start with \"/\"."
  }
}

variable "image_tag" {
  type        = string
  description = "Tag of the image in the app's ECR repository to run. Tags are immutable, so deploying a new version means pushing a new tag and changing this."
}

variable "cpu" {
  type        = number
  description = "CPU units per container (1024 is one vCPU). Must form a valid Fargate pair with memory: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/fargate-tasks-services.html#fargate-tasks-size"
  default     = 256
}

variable "memory" {
  type        = number
  description = "Memory per container, in MiB. Must form a valid Fargate pair with cpu."
  default     = 512
}

variable "desired_count" {
  type        = number
  description = "How many copies of the app to run. Two or more spread across availability zones, so one zone failing doesn't take the app down."
  default     = 1
}

variable "database" {
  type = object({
    address           = string
    port              = number
    name              = string
    secret_arn        = string
    security_group_id = string
  })
  description = "The database module's outputs, or null for an app without a database. The app gets the connection details as the DB_HOST, DB_PORT, DB_NAME, DB_USERNAME and DB_PASSWORD environment variables."
  default     = null
}
