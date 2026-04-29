# ElastiCache Redis, single node, cache.t3.micro.
#
# Why aws_elasticache_cluster (not aws_elasticache_replication_group):
# replication_group is the cluster-mode/HA resource and forces
# multi-node. We want a single t3.micro for dev — the cluster
# resource with num_cache_nodes=1 is the only way to get that.
#
# Why no transit_encryption_enabled: Redis without encryption uses
# port 6379 (plain TCP); enabling it requires port 6380 + TLS plus
# rebuilding the Havoc Redis client to dial TLS. That's a Phase 7
# concern. Within the VPC, plain Redis is reachable only via the SG
# below.
#
# Why no auth_token: same reason — plain Redis. The lock + killswitch
# semantics don't need auth on a private network.

resource "aws_elasticache_subnet_group" "this" {
  name       = "${var.prefix}-redis"
  subnet_ids = [
    for s in aws_subnet.public : s.id if s.availability_zone == local.primary_az
  ]
}

resource "aws_security_group" "redis" {
  name        = "${var.prefix}-redis-sg"
  description = "Allow Redis access from the EKS cluster shared SG only."
  vpc_id      = aws_vpc.this.id
}

resource "aws_security_group_rule" "redis_from_eks_nodes" {
  type                     = "ingress"
  from_port                = 6379
  to_port                  = 6379
  protocol                 = "tcp"
  security_group_id        = aws_security_group.redis.id
  source_security_group_id = aws_eks_cluster.this.vpc_config[0].cluster_security_group_id
  description              = "Redis ingress from EKS cluster SG"
}

resource "aws_security_group_rule" "redis_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.redis.id
  description       = "All egress (cosmetic - Redis does not initiate connections)"
}

resource "aws_elasticache_cluster" "this" {
  cluster_id           = local.redis_id
  engine               = "redis"
  engine_version       = var.redis_engine_version
  node_type            = var.redis_node_type
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7"
  port                 = 6379

  subnet_group_name  = aws_elasticache_subnet_group.this.name
  security_group_ids = [aws_security_group.redis.id]
}
