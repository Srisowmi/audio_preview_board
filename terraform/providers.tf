terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  # Local state for simplicity (terraform.tfstate is git-ignored).
}

provider "aws" {
  region = var.aws_region

  # Credentials come from the standard AWS chain (env vars, ~/.aws, SSO).
  # Nothing secret lives in this repository.
  default_tags {
    tags = {
      Project     = var.app_name
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}
