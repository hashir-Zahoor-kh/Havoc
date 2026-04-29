# Locals + data sources shared across the rest of the module.

data "aws_caller_identity" "current" {}

locals {
  cluster_name = "${var.prefix}-eks"
  rds_id       = "${var.prefix}-pg"
  redis_id     = "${var.prefix}-redis"

  # The single AZ that actually hosts workloads. The other AZs in
  # var.azs exist only to satisfy EKS / RDS subnet-group minimums.
  primary_az = var.azs[0]

  account_id = data.aws_caller_identity.current.account_id
}
