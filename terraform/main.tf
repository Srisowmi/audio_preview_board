# Backend infrastructure: ECR repository + Lambda (container image) + public Function URL.
#
# Chicken-and-egg: a container-image Lambda cannot be created until its image
# exists in ECR, and the image is built by GitHub Actions (no local Docker). So
# apply in two steps:
#   1. terraform apply -target=aws_ecr_repository.api -target=aws_iam_role_policy.github_actions
#      (creates the ECR repo plus the GitHub OIDC provider and CI role)
#      then save the `github_actions_role_arn` output as the GitHub secret AWS_ROLE_ARN
#   2. push to main: the workflow tests, builds and pushes the first image
#      (it skips the Lambda update because the function does not exist yet)
#   3. terraform apply                                     (creates everything else)
# After that, CI deploys new images with `aws lambda update-function-code`;
# Terraform ignores image_uri changes so it never reverts a CI deploy.

locals {
  name = "${var.app_name}-${var.environment}"
}

# ---------------------------------------------------------------- ECR

resource "aws_ecr_repository" "api" {
  name                 = local.name
  image_tag_mutability = "MUTABLE" # CI also moves the "latest" tag
  force_delete         = true      # lets `terraform destroy` remove a repo that still has images

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "api" {
  repository = aws_ecr_repository.api.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep the 5 most recent images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 5
      }
      action = { type = "expire" }
    }]
  })
}

# ---------------------------------------------------------------- Logs

resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/lambda/${local.name}"
  retention_in_days = var.log_retention_days
}

# ---------------------------------------------------------------- IAM

data "aws_iam_policy_document" "lambda_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "lambda" {
  name               = "${local.name}-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
}

# Least privilege: the function may only write to its own log group.
data "aws_iam_policy_document" "lambda_logs" {
  statement {
    actions   = ["logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["${aws_cloudwatch_log_group.api.arn}:*"]
  }
}

resource "aws_iam_role_policy" "lambda_logs" {
  name   = "logs"
  role   = aws_iam_role.lambda.id
  policy = data.aws_iam_policy_document.lambda_logs.json
}

# ---------------------------------------------------------------- Lambda

resource "aws_lambda_function" "api" {
  function_name = local.name
  role          = aws_iam_role.lambda.arn
  package_type  = "Image"
  image_uri     = "${aws_ecr_repository.api.repository_url}:${var.image_tag}"
  architectures = ["x86_64"]
  memory_size   = var.lambda_memory_mb
  timeout       = var.lambda_timeout_seconds

  environment {
    variables = {
      CACHE_TTL      = var.cache_ttl
      ALLOWED_ORIGIN = var.allowed_origin
      PORT           = tostring(var.app_port)
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.api,
    aws_iam_role_policy.lambda_logs,
  ]

  lifecycle {
    # CI rolls out new images with update-function-code; don't fight it.
    ignore_changes = [image_uri]
  }
}

# Public HTTPS endpoint. CORS is handled inside the Go app, so none is set here
# (setting it in both places would send duplicate CORS headers).
resource "aws_lambda_function_url" "api" {
  function_name      = aws_lambda_function.api.function_name
  authorization_type = "NONE"
}

# Public Function URLs need both permissions.
resource "aws_lambda_permission" "url_invoke" {
  statement_id           = "AllowPublicFunctionUrl"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.api.function_name
  principal              = "*"
  function_url_auth_type = "NONE"
}

resource "aws_lambda_permission" "url_invoke_function" {
  statement_id             = "AllowPublicInvokeViaFunctionUrl"
  action                   = "lambda:InvokeFunction"
  function_name            = aws_lambda_function.api.function_name
  principal                = "*"
  invoked_via_function_url = true
}
