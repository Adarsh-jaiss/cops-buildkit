terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "4.49.0"
    }

    helm = {
      source  = "hashicorp/helm"
      version = "2.6.0"
    }
  }
  backend "s3" {
    bucket         = "cops-production-ap-south-1"
    key            = "production"
    region         = "ap-south-1"
    encrypt        = true
    dynamodb_table = "cops-production-ap-south-1"
  }

  required_version = ">= 1.2.0"
}