# RDS Postgres. Single-AZ, db.t3.micro, 20 GiB gp3.
#
# Why not publicly accessible: the SG below only lets the EKS
# cluster's shared SG hit port 5432. To reach the DB from the
# laptop, exec into a pod (`kubectl run -it --rm psql --image
# postgres:16 -- psql ...`) — same pattern as `kubectl exec` into
# the local kind-hosted Postgres pod.
#
# Why subnet group across both AZs even though the instance lives
# in one: RDS *requires* DB subnet groups to span ≥2 AZs even for
# single-AZ deployments. The instance's `availability_zone` is
# pinned to local.primary_az, so the second subnet is dead capacity
# kept around purely to satisfy the API.
#
# Password is generated, never in source. Fetch post-apply with
# `terraform output -raw rds_admin_password`.

resource "random_password" "rds" {
  length  = 24
  special = true
  # RDS Postgres rejects '/', '@', '"', and ' ' in admin passwords.
  override_special = "!#$%^&*()-_=+[]{}<>?"
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.prefix}-rds"
  subnet_ids = aws_subnet.public[*].id
}

resource "aws_security_group" "rds" {
  name        = "${var.prefix}-rds-sg"
  description = "Allow Postgres access from the EKS cluster shared SG only."
  vpc_id      = aws_vpc.this.id
}

resource "aws_security_group_rule" "rds_from_eks_nodes" {
  type                     = "ingress"
  from_port                = 5432
  to_port                  = 5432
  protocol                 = "tcp"
  security_group_id        = aws_security_group.rds.id
  source_security_group_id = aws_eks_cluster.this.vpc_config[0].cluster_security_group_id
  description              = "Postgres ingress from EKS cluster SG"
}

resource "aws_security_group_rule" "rds_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.rds.id
  description       = "All egress (RDS does not initiate connections, so this is mostly cosmetic)"
}

resource "aws_db_instance" "this" {
  identifier        = local.rds_id
  engine            = "postgres"
  engine_version    = var.rds_engine_version
  instance_class    = var.rds_instance_class
  allocated_storage = var.rds_allocated_storage
  storage_type      = "gp3"
  storage_encrypted = true

  db_name  = var.rds_database_name
  username = var.rds_admin_user
  password = random_password.rds.result

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  availability_zone      = local.primary_az
  multi_az               = false
  publicly_accessible    = false

  # Dev: skip final snapshot and zero out backups. Flip both for prod.
  skip_final_snapshot     = true
  backup_retention_period = 0
  delete_automated_backups = true

  # Apply minor version bumps in the next maintenance window. Saves
  # the manual upgrade dance for routine patches.
  auto_minor_version_upgrade = true

  # Critical for `terraform destroy` to actually remove the instance
  # without a manual confirmation step.
  deletion_protection = false
}
