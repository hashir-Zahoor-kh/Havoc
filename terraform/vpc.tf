# Minimal VPC for dev. Public subnets only — no NAT Gateway, no
# private subnets. Worker nodes get public IPs and reach the
# internet directly via the IGW. This saves ~$33/mo per NAT GW;
# the tradeoff is that nodes are attackable from the internet at
# the IP layer, but the security groups (in eks.tf, rds.tf,
# elasticache.tf) lock all inbound traffic to either the cluster
# or specific service-to-service paths.
#
# Two subnets across two AZs are required by EKS (control plane)
# and RDS (DB subnet group). Only the first AZ hosts workloads —
# see local.primary_az.

resource "aws_vpc" "this" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.prefix}-vpc"
  }
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id

  tags = {
    Name = "${var.prefix}-igw"
  }
}

resource "aws_subnet" "public" {
  count                   = length(var.azs)
  vpc_id                  = aws_vpc.this.id
  cidr_block              = var.public_subnet_cidrs[count.index]
  availability_zone       = var.azs[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.prefix}-public-${var.azs[count.index]}"
    # EKS uses these tags to discover which subnets a public ELB
    # may be placed into. Harmless when no ELB exists.
    "kubernetes.io/role/elb" = "1"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.this.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.this.id
  }

  tags = {
    Name = "${var.prefix}-public-rt"
  }
}

resource "aws_route_table_association" "public" {
  count          = length(aws_subnet.public)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}
