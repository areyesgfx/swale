# The excluded zones are the few where CloudFront VPC origins, which the
# service module uses to reach its internal load balancer, aren't
# supported: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-vpc-origins.html#vpc-origins-supported-regions
data "aws_availability_zones" "available" {
  state            = "available"
  exclude_zone_ids = ["use1-az3", "usw1-az2", "apne1-az3", "cac1-az3"]
}

data "aws_region" "current" {}

locals {
  azs = slice(data.aws_availability_zones.available.names, 0, var.az_count)

  # NAT gateways: one shared gateway in "single" mode, one per AZ in
  # "per_az" mode. Each private route table (one per AZ, so per-AZ traffic
  # can stay in-AZ) picks its target NAT gateway from this list via
  # nat_gateway_index below.
  nat_gateway_count = var.nat_mode == "per_az" ? var.az_count : 1
}

resource "aws_vpc" "this" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = { Name = var.name }
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id

  tags = { Name = var.name }
}

# Public subnets take the first az_count /20-or-smaller chunks of the VPC
# CIDR, private subnets the next az_count chunks - see cidrsubnet newbits
# note on vpc_cidr's description.
resource "aws_subnet" "public" {
  count                   = var.az_count
  vpc_id                  = aws_vpc.this.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 4, count.index)
  availability_zone       = local.azs[count.index]
  map_public_ip_on_launch = true

  tags = { Name = "${var.name}-public-${local.azs[count.index]}" }
}

resource "aws_subnet" "private" {
  count             = var.az_count
  vpc_id            = aws_vpc.this.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 4, var.az_count + count.index)
  availability_zone = local.azs[count.index]
  # map_public_ip_on_launch defaults to false: private subnets get no
  # public IPs, the "private subnets by default" secure default.

  tags = { Name = "${var.name}-private-${local.azs[count.index]}" }
}

resource "aws_eip" "nat" {
  count  = local.nat_gateway_count
  domain = "vpc"

  tags = { Name = "${var.name}-nat-${local.azs[count.index]}" }
}

resource "aws_nat_gateway" "this" {
  count         = local.nat_gateway_count
  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id
  depends_on    = [aws_internet_gateway.this]

  tags = { Name = "${var.name}-${local.azs[count.index]}" }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.this.id

  tags = { Name = "${var.name}-public" }
}

resource "aws_route" "public_internet" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.this.id
}

resource "aws_route_table_association" "public" {
  count          = var.az_count
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# One private route table per AZ (not one shared table) even in "single"
# NAT mode, so per-AZ tables already exist if a project switches to
# "per_az" later - only the NAT gateway target below changes.
resource "aws_route_table" "private" {
  count  = var.az_count
  vpc_id = aws_vpc.this.id

  tags = { Name = "${var.name}-private-${local.azs[count.index]}" }
}

resource "aws_route" "private_nat" {
  count                  = var.az_count
  route_table_id         = aws_route_table.private[count.index].id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.this[var.nat_mode == "per_az" ? count.index : 0].id
}

resource "aws_route_table_association" "private" {
  count          = var.az_count
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}

# Gateway endpoint: no hourly charge (unlike an interface endpoint), and
# keeps S3 traffic from private subnets off the public internet path
# entirely rather than routing it through NAT.
resource "aws_vpc_endpoint" "s3" {
  vpc_id            = aws_vpc.this.id
  service_name      = "com.amazonaws.${data.aws_region.current.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = concat(aws_route_table.private[*].id, [aws_route_table.public.id])

  tags = { Name = "${var.name}-s3" }
}

# Least-privilege starting point: no ingress rules at all (AWS security
# groups deny all inbound by default), all outbound allowed. Open specific
# ports via additional security groups as later phases need them.
resource "aws_security_group" "baseline" {
  name_prefix = "${var.name}-baseline-"
  description = "Baseline security group: no inbound access, all outbound allowed."
  vpc_id      = aws_vpc.this.id

  egress {
    description = "Allow all outbound traffic."
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.name}-baseline" }

  lifecycle {
    create_before_destroy = true
  }
}
