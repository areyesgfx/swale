data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

locals {
  web            = var.port != null
  container_name = "app"
}

# Tags can't be overwritten, so a tag always means the same image and a
# deploy can be rolled back by setting image_tag to the previous one.
resource "aws_ecr_repository" "this" {
  name                 = var.name
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "this" {
  repository = aws_ecr_repository.this.name
  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep the 30 most recent images."
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 30
      }
      action = { type = "expire" }
    }]
  })
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/ecs/${var.name}"
  retention_in_days = 30
}

# Both roles can only be assumed by ECS tasks in this account and region,
# which stops another account's ECS from using them (the "confused
# deputy" problem).
data "aws_iam_policy_document" "assume_by_ecs_tasks" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [data.aws_caller_identity.current.account_id]
    }

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = ["arn:aws:ecs:${data.aws_region.current.region}:${data.aws_caller_identity.current.account_id}:*"]
    }
  }
}

# The execution role is what ECS itself uses to start the container: pull
# its image and send its logs. It's scoped to this app's repository and
# log group, rather than the AWS-managed AmazonECSTaskExecutionRolePolicy,
# which allows pulling from every repository in the account.
resource "aws_iam_role" "execution" {
  name_prefix        = "${var.name}-exec-"
  assume_role_policy = data.aws_iam_policy_document.assume_by_ecs_tasks.json
}

data "aws_iam_policy_document" "execution" {
  statement {
    sid       = "AuthenticateToECR"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }

  statement {
    sid = "PullAppImage"
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:GetDownloadUrlForLayer",
    ]
    resources = [aws_ecr_repository.this.arn]
  }

  statement {
    sid       = "WriteAppLogs"
    actions   = ["logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["${aws_cloudwatch_log_group.this.arn}:*"]
  }
}

resource "aws_iam_role_policy" "execution" {
  name   = "pull-image-and-write-logs"
  role   = aws_iam_role.execution.id
  policy = data.aws_iam_policy_document.execution.json
}

# The task role is what the app's own code runs as. It starts with no
# permissions; attach policies to it as the app needs AWS services.
resource "aws_iam_role" "task" {
  name_prefix        = "${var.name}-task-"
  assume_role_policy = data.aws_iam_policy_document.assume_by_ecs_tasks.json
}

resource "aws_ecs_cluster" "this" {
  name = var.name
}

resource "aws_ecs_task_definition" "this" {
  family                   = var.name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.cpu
  memory                   = var.memory
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  # Images must be built for this architecture: on an Apple silicon Mac,
  # that means `docker build --platform linux/amd64`.
  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }

  container_definitions = jsonencode([{
    name         = local.container_name
    image        = "${aws_ecr_repository.this.repository_url}:${var.image_tag}"
    essential    = true
    portMappings = local.web ? [{ containerPort = var.port, protocol = "tcp" }] : []
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.this.name
        awslogs-region        = data.aws_region.current.region
        awslogs-stream-prefix = local.container_name
      }
    }
  }])
}

resource "aws_security_group" "service" {
  name_prefix = "${var.name}-service-"
  description = "The app's containers: inbound only from its load balancer, all outbound."
  vpc_id      = var.vpc_id

  tags = { Name = "${var.name}-service" }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_egress_rule" "service_all" {
  security_group_id = aws_security_group.service.id
  description       = "Allow all outbound traffic, e.g. to pull the image and call other services."
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "service_from_lb" {
  count = local.web ? 1 : 0

  security_group_id            = aws_security_group.service.id
  description                  = "Web requests forwarded by the load balancer."
  referenced_security_group_id = aws_security_group.lb[0].id
  ip_protocol                  = "tcp"
  from_port                    = var.port
  to_port                      = var.port
}

resource "aws_security_group" "lb" {
  count = local.web ? 1 : 0

  name_prefix = "${var.name}-lb-"
  description = "The app's load balancer: HTTP from anywhere, outbound only to the app's containers."
  vpc_id      = var.vpc_id

  tags = { Name = "${var.name}-lb" }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_ingress_rule" "lb_http" {
  count = local.web ? 1 : 0

  security_group_id = aws_security_group.lb[0].id
  description       = "HTTP from anywhere."
  ip_protocol       = "tcp"
  from_port         = 80
  to_port           = 80
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "lb_to_service" {
  count = local.web ? 1 : 0

  security_group_id            = aws_security_group.lb[0].id
  description                  = "Forward requests to the app's containers."
  referenced_security_group_id = aws_security_group.service.id
  ip_protocol                  = "tcp"
  from_port                    = var.port
  to_port                      = var.port
}

resource "aws_lb" "this" {
  count = local.web ? 1 : 0

  name                       = var.name
  load_balancer_type         = "application"
  internal                   = false
  security_groups            = [aws_security_group.lb[0].id]
  subnets                    = var.public_subnet_ids
  drop_invalid_header_fields = true
}

# No name, so Terraform generates a unique one: changing the port replaces
# the target group, and create_before_destroy needs the old and new ones
# to exist side by side.
resource "aws_lb_target_group" "this" {
  count = local.web ? 1 : 0

  port                 = var.port
  protocol             = "HTTP"
  target_type          = "ip"
  vpc_id               = var.vpc_id
  deregistration_delay = 30

  health_check {
    path                = var.health_check_path
    matcher             = "200-399"
    interval            = 15
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }

  tags = { Name = var.name }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_lb_listener" "http" {
  count = local.web ? 1 : 0

  load_balancer_arn = aws_lb.this[0].arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.this[0].arn
  }
}

resource "aws_ecs_service" "this" {
  name            = var.name
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.this.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"
  propagate_tags  = "SERVICE"

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.service.id]
    assign_public_ip = false
  }

  # A deploy whose containers keep failing to start is stopped and rolled
  # back to the last version that worked, instead of retrying forever.
  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  dynamic "load_balancer" {
    for_each = local.web ? [1] : []
    content {
      target_group_arn = aws_lb_target_group.this[0].arn
      container_name   = local.container_name
      container_port   = var.port
    }
  }

  health_check_grace_period_seconds = local.web ? 60 : null

  # A target group can't be attached to a service until it's attached to a
  # load balancer.
  depends_on = [aws_lb_listener.http]
}
