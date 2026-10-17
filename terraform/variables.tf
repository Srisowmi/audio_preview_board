variable "aws_region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "ap-south-1"
}

variable "environment" {
  description = "Environment name, used in resource names and tags."
  type        = string
  default     = "prod"
}

variable "app_name" {
  description = "Application name, used as a prefix for resource names."
  type        = string
  default     = "audio-preview"
}

variable "app_port" {
  description = "Port the Go server listens on when run as a normal server (PORT env var). Ignored by Lambda, which is invoked through the Function URL."
  type        = number
  default     = 8080
}

variable "cache_ttl" {
  description = "In-memory cache lifetime as a Go duration (e.g. 30m, 1h)."
  type        = string
  default     = "1h"

  validation {
    condition     = can(regex("^([0-9]+(\\.[0-9]+)?(ns|us|ms|s|m|h))+$", var.cache_ttl))
    error_message = "cache_ttl must be a Go duration such as 30m or 1h."
  }
}

variable "allowed_origin" {
  description = "Origin allowed by the API's CORS middleware, e.g. https://my-app.vercel.app (no trailing slash)."
  type        = string
  default     = "http://localhost:5173"

  validation {
    condition     = !endswith(var.allowed_origin, "/")
    error_message = "allowed_origin must not end with a slash."
  }
}

variable "github_repo" {
  description = "GitHub repository (owner/name) whose main branch may deploy via OIDC."
  type        = string
  default     = "Srisowmi/audio_preview_board"
}

variable "create_github_oidc_provider" {
  description = "Create the GitHub OIDC identity provider. Set false if your AWS account already has one for token.actions.githubusercontent.com."
  type        = bool
  default     = true
}

variable "image_tag" {
  description = "ECR image tag used when the Lambda function is first created. CI updates the running image afterwards."
  type        = string
  default     = "latest"
}

variable "lambda_memory_mb" {
  description = "Lambda memory size in MB."
  type        = number
  default     = 256
}

variable "lambda_timeout_seconds" {
  description = "Lambda timeout in seconds. Must exceed the 5s upstream iTunes timeout."
  type        = number
  default     = 10
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days."
  type        = number
  default     = 14
}
