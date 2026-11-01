output "repository_url" {
  description = "ECR repository to push the app's images to."
  value       = aws_ecr_repository.this.repository_url
}

output "url" {
  description = "Where the app is reachable, or null for a background worker."
  value       = local.web ? "https://${aws_cloudfront_distribution.this[0].domain_name}" : null
}

output "cluster_name" {
  value = aws_ecs_cluster.this.name
}

output "service_name" {
  value = aws_ecs_service.this.name
}

output "task_role_name" {
  description = "IAM role the app runs as. Attach policies to it to let the app use other AWS services."
  value       = aws_iam_role.task.name
}

output "log_group_name" {
  value = aws_cloudwatch_log_group.this.name
}
