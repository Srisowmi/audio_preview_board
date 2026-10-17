output "api_url" {
  description = "Public HTTPS base URL of the backend (use as VITE_API_URL)."
  value       = trimsuffix(aws_lambda_function_url.api.function_url, "/")
}

output "ecr_repository_url" {
  description = "ECR repository URL that CI pushes images to."
  value       = aws_ecr_repository.api.repository_url
}

output "ecr_repository_name" {
  description = "ECR repository name (GitHub Actions variable ECR_REPOSITORY)."
  value       = aws_ecr_repository.api.name
}

output "github_actions_role_arn" {
  description = "IAM role for GitHub Actions (store as the GitHub secret AWS_ROLE_ARN)."
  value       = aws_iam_role.github_actions.arn
}

output "lambda_function_name" {
  description = "Lambda function name (GitHub Actions variable LAMBDA_FUNCTION_NAME)."
  value       = aws_lambda_function.api.function_name
}

output "log_group_name" {
  description = "CloudWatch log group for the API."
  value       = aws_cloudwatch_log_group.api.name
}
