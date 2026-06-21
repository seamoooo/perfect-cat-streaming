terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.50"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
    newrelic = {
      source  = "newrelic/newrelic"
      version = "~> 3.40"
    }
  }

  # Uncomment to use S3 remote state.
  # backend "s3" {
  #   bucket         = "your-tfstate-bucket"
  #   key            = "perfect-cat-streaming/terraform.tfstate"
  #   region         = "ap-northeast-1"
  #   dynamodb_table = "your-tflock-table"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      Project     = var.project
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# CloudFront uses ACM certs from us-east-1 only.
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
  default_tags {
    tags = {
      Project     = var.project
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# New Relic — used for alert policies/conditions and notification workflows.
# Authenticates with the User key (NRAK-*). When new_relic_user_api_key is empty
# the provider is never invoked because every newrelic resource is gated on
# local.nr_alerts_enabled (count = 0), so plans still work without the key.
provider "newrelic" {
  account_id = var.new_relic_account_id
  api_key    = var.new_relic_user_api_key
  region     = var.new_relic_region
}
