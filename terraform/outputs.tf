# Outputs consumed by Helm in Phase 6b and by the operator at the
# command line. Sensitive values (RDS password) are flagged so they
# don't leak into terraform.tfstate dumps or CI logs; fetch them
# with `terraform output -raw <name>`.

output "region" {
  value       = var.region
  description = "AWS region everything lives in."
}

output "account_id" {
  value       = local.account_id
  description = "AWS account ID."
}

output "eks_cluster_name" {
  value       = aws_eks_cluster.this.name
  description = "EKS cluster name."
}

output "eks_kubeconfig_cmd" {
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${aws_eks_cluster.this.name}"
  description = "One-liner to merge the kubeconfig for this cluster."
}

output "eks_cluster_endpoint" {
  value       = aws_eks_cluster.this.endpoint
  description = "EKS API server endpoint."
}

output "ecr_registry" {
  value       = "${local.account_id}.dkr.ecr.${var.region}.amazonaws.com"
  description = "ECR registry hostname for the account."
}

output "ecr_repository_urls" {
  value       = { for k, r in aws_ecr_repository.this : k => r.repository_url }
  description = "Map of binary name → full ECR repo URL. Used as image refs in Helm values."
}

output "rds_endpoint" {
  value       = aws_db_instance.this.address
  description = "Postgres host. Port is the default 5432."
}

output "rds_port" {
  value       = aws_db_instance.this.port
  description = "Postgres port."
}

output "rds_database" {
  value       = aws_db_instance.this.db_name
  description = "Database name created at instance launch."
}

output "rds_admin_user" {
  value       = aws_db_instance.this.username
  description = "Postgres admin username."
}

output "rds_admin_password" {
  value       = random_password.rds.result
  description = "Postgres admin password. Fetch with `terraform output -raw rds_admin_password`."
  sensitive   = true
}

output "redis_endpoint" {
  value       = aws_elasticache_cluster.this.cache_nodes[0].address
  description = "Redis host. Port is 6379 (plain TCP, no auth — VPC-private)."
}

output "redis_port" {
  value       = aws_elasticache_cluster.this.cache_nodes[0].port
  description = "Redis port."
}
