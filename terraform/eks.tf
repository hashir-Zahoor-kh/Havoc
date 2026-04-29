# EKS cluster + a single managed node group.
#
# Why managed node group (not self-managed, not Fargate, not Auto
# Mode): managed node groups are the path of least friction for a
# small dev cluster — AWS owns the launch template, the AMI
# rotation, and the rolling-update on version bumps. Auto Mode
# (Karpenter-based) would scale to zero between sessions but adds
# a layer of indirection that's overkill for two t3.small nodes
# we're going to tear down nightly anyway.
#
# Why t3.small (not t3.medium): the new-account first-launch
# restriction on this AWS account blocks non-free-tier instance
# types until either a manual t3.micro launch clears it or AWS
# raises the implicit quota. t3.small (2 vCPU / 2 GiB) fits AL2023
# + system pods + Havoc DaemonSets with ~500 MiB headroom; tight
# enough that sustained load could trigger kubelet eviction. Path
# out is t3.medium once the account restriction lifts.
#
# Why pinned to azs[0] only: the cluster control plane has to
# advertise subnets in two AZs, but the workers are constrained
# to one. That keeps node-to-RDS, node-to-Redis, and node-to-node
# traffic all in-AZ (no $0.01/GB cross-AZ charge).
#
# Auth uses Access Entries (the modern API), not the legacy
# aws-auth ConfigMap — `authentication_mode = "API"` switches the
# cluster to entries-only. The IAM user that ran `terraform apply`
# (var.admin_iam_user_arn) gets cluster-admin so `kubectl` works
# from the laptop without a separate `aws-auth` edit.

# ---------- IAM: cluster role ----------

resource "aws_iam_role" "eks_cluster" {
  name = "${var.prefix}-eks-cluster-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "eks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "eks_cluster_policy" {
  role       = aws_iam_role.eks_cluster.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
}

# ---------- IAM: node role ----------
#
# AmazonEKS_CNI_Policy is needed by the VPC CNI plugin to manage ENIs.
# AmazonEC2ContainerRegistryReadOnly lets nodes pull from ECR without
# a separate registry secret.

resource "aws_iam_role" "eks_node" {
  name = "${var.prefix}-eks-node-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "eks_node_worker" {
  role       = aws_iam_role.eks_node.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
}

resource "aws_iam_role_policy_attachment" "eks_node_cni" {
  role       = aws_iam_role.eks_node.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
}

resource "aws_iam_role_policy_attachment" "eks_node_ecr" {
  role       = aws_iam_role.eks_node.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

# ---------- Cluster ----------

resource "aws_eks_cluster" "this" {
  name     = local.cluster_name
  role_arn = aws_iam_role.eks_cluster.arn
  version  = var.eks_cluster_version

  vpc_config {
    subnet_ids              = aws_subnet.public[*].id
    endpoint_public_access  = true
    endpoint_private_access = true
  }

  access_config {
    authentication_mode                         = "API"
    bootstrap_cluster_creator_admin_permissions = false
  }

  # No log_types specified = no CloudWatch logs. Saves ~$1-3/mo;
  # for prod, enable api+audit+authenticator at minimum.

  depends_on = [
    aws_iam_role_policy_attachment.eks_cluster_policy,
  ]
}

# ---------- Access Entries ----------
#
# Two entries: one for the operator (the admin IAM user/role
# specified in var.admin_iam_user_arn) and one for the node role
# (so nodes can join — required when bootstrap_cluster_creator_admin
# is false).

resource "aws_eks_access_entry" "admin" {
  cluster_name  = aws_eks_cluster.this.name
  principal_arn = var.admin_iam_user_arn
  type          = "STANDARD"
}

resource "aws_eks_access_policy_association" "admin" {
  cluster_name  = aws_eks_cluster.this.name
  principal_arn = var.admin_iam_user_arn
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"

  access_scope {
    type = "cluster"
  }

  depends_on = [aws_eks_access_entry.admin]
}

# ---------- Managed node group ----------

resource "aws_eks_node_group" "system" {
  cluster_name    = aws_eks_cluster.this.name
  node_group_name = "system"
  node_role_arn   = aws_iam_role.eks_node.arn
  # subnet_ids restricted to the primary AZ's subnet — the cluster
  # itself spans both AZs, but nodes only land in one.
  subnet_ids = [
    for s in aws_subnet.public : s.id if s.availability_zone == local.primary_az
  ]

  instance_types = [var.eks_node_instance_type]
  ami_type       = "AL2023_x86_64_STANDARD"
  disk_size      = var.eks_node_disk_size
  capacity_type  = "ON_DEMAND"

  scaling_config {
    desired_size = var.eks_node_count
    min_size     = var.eks_node_count
    max_size     = var.eks_node_count
  }

  update_config {
    max_unavailable = 1
  }

  depends_on = [
    aws_iam_role_policy_attachment.eks_node_worker,
    aws_iam_role_policy_attachment.eks_node_cni,
    aws_iam_role_policy_attachment.eks_node_ecr,
  ]
}
