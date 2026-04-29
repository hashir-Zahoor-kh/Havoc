# Inputs. Defaults match the dev environment — overrideable per-env
# via `-var-file=dev.tfvars` (or prod.tfvars when that exists).

variable "region" {
  description = "AWS region for all resources."
  type        = string
  default     = "us-east-1"
}

variable "prefix" {
  description = "Resource name prefix. Keeps names disambiguated if multiple Havoc envs share an account."
  type        = string
  default     = "havoc"
}

variable "tags" {
  description = "Tags applied to every resource via the provider's default_tags."
  type        = map(string)
  default = {
    project = "havoc"
    env     = "dev"
    owner   = "hashir"
  }
}

# ---------- VPC / subnets ----------

variable "vpc_cidr" {
  description = "VPC CIDR. /16 leaves room for many subnets if dev grows."
  type        = string
  default     = "10.0.0.0/16"
}

variable "azs" {
  description = "AZs the subnets are placed in. EKS requires ≥2; data-plane resources are pinned to azs[0] only."
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

variable "public_subnet_cidrs" {
  description = "Public subnet CIDRs, one per entry in var.azs. Two are required by EKS even though only the first hosts workloads."
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
}

# ---------- EKS ----------

variable "eks_cluster_version" {
  description = "EKS Kubernetes version. Pin so plans are deterministic; bump explicitly when you intend to upgrade."
  type        = string
  default     = "1.30"
}

variable "eks_node_instance_type" {
  description = "Worker node EC2 type. t3.small = 2 vCPU / 2 GiB — RAM is tight but fits AL2023 + system pods + the Havoc DaemonSets with ~500 MiB headroom per node. t3.medium would be more comfortable but is blocked by the new-account first-launch restriction on this AWS account."
  type        = string
  default     = "t3.small"
}

variable "eks_node_count" {
  description = "Managed node group desired/min/max. 2 lets a DaemonSet land on more than one node without multi-AZ cost."
  type        = number
  default     = 2
}

variable "eks_node_disk_size" {
  description = "Worker root EBS size in GiB. 20 is the EKS minimum AL2023 image will install on; bumping to 30 gives breathing room for image cache."
  type        = number
  default     = 20
}

variable "admin_iam_user_arn" {
  description = "IAM user/role granted EKS cluster-admin via Access Entry. Set to whoever runs `aws eks update-kubeconfig` from the laptop."
  type        = string
}

# ---------- RDS ----------

variable "rds_engine_version" {
  description = "Postgres major version. 16 matches the local compose stack so migrations behave identically."
  type        = string
  default     = "16"
}

variable "rds_instance_class" {
  description = "RDS instance class. db.t3.micro is the cheapest that supports Postgres 16."
  type        = string
  default     = "db.t3.micro"
}

variable "rds_allocated_storage" {
  description = "RDS storage in GiB. 20 is the minimum; gp3 storage type is cheaper than gp2 at this size."
  type        = number
  default     = 20
}

variable "rds_admin_user" {
  description = "Postgres admin username."
  type        = string
  default     = "havocadmin"
}

variable "rds_database_name" {
  description = "Database created at instance launch. Saves a `CREATE DATABASE` step in migrations."
  type        = string
  default     = "havoc"
}

# ---------- ElastiCache ----------

variable "redis_node_type" {
  description = "ElastiCache node type. cache.t3.micro is free-tier eligible for new accounts and the cheapest otherwise."
  type        = string
  default     = "cache.t3.micro"
}

variable "redis_engine_version" {
  description = "Redis major.minor. 7.1 is the latest stable Redis OSS branch ElastiCache supports."
  type        = string
  default     = "7.1"
}

# ---------- ECR ----------

variable "ecr_repositories" {
  description = "ECR repos to create. One per Havoc binary."
  type        = list(string)
  default     = ["havoc-control", "havoc-agent", "havoc-recorder"]
}
