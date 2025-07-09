terraform {
  required_version = ">= 1.0"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.0"
    }
  }
}

# Local file resources for testing
resource "local_file" "config" {
  filename = "${path.module}/config.txt"
  content  = "Environment: production\nVersion: 1.0.0"
}

resource "local_file" "data" {
  filename = "${path.module}/data.txt"
  content  = "Sample data file"
}

# Local sensitive file
resource "local_sensitive_file" "secret" {
  filename = "${path.module}/secret.txt"
  content  = "super-secret-value"
}

# Variables
variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "app_version" {
  description = "Application version"
  type        = string
  default     = "1.0.0"
}

# Outputs
output "config_path" {
  value = local_file.config.filename
}

output "environment" {
  value = var.environment
}