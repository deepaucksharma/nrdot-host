output "db_instance_id" {
  description = "The RDS instance ID"
  value       = try(aws_db_instance.main[0].id, "")
}

output "db_instance_arn" {
  description = "The ARN of the RDS instance"
  value       = try(aws_db_instance.main[0].arn, "")
}

output "db_instance_address" {
  description = "The address of the RDS instance"
  value       = try(aws_db_instance.main[0].address, "")
}

output "db_instance_endpoint" {
  description = "The connection endpoint"
  value       = try(aws_db_instance.main[0].endpoint, "")
}

output "db_instance_port" {
  description = "The database port"
  value       = try(aws_db_instance.main[0].port, var.port)
}

output "db_subnet_group_name" {
  description = "The name of the subnet group"
  value       = aws_db_subnet_group.main.name
}

output "db_parameter_group_name" {
  description = "The name of the parameter group"
  value       = aws_db_parameter_group.main.name
}

output "db_security_group_id" {
  description = "The security group ID of the RDS instance"
  value       = aws_security_group.rds.id
}

output "db_instance_name" {
  description = "The database name"
  value       = try(aws_db_instance.main[0].db_name, var.database_name)
}

output "replica_endpoints" {
  description = "The connection endpoints for read replicas"
  value       = [for r in aws_db_instance.replica : r.endpoint]
}

output "replica_addresses" {
  description = "The addresses of read replicas"
  value       = [for r in aws_db_instance.replica : r.address]
}