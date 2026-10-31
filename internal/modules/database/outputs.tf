output "address" {
  value = aws_db_instance.this.address
}

output "port" {
  value = aws_db_instance.this.port
}

output "name" {
  description = "Name of the database the app connects to."
  value       = aws_db_instance.this.db_name
}

output "secret_arn" {
  description = "Secrets Manager secret holding the master user's username and password, as JSON."
  value       = aws_secretsmanager_secret.master.arn
}

output "security_group_id" {
  value = aws_security_group.this.id
}
