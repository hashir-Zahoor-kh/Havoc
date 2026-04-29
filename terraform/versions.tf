# Terraform + provider versions and remote state.
#
# Remote state lives in a hand-created S3 bucket
# (havoc-tfstate-802531653822) with versioning + AES256 encryption.
# DynamoDB table (havoc-tfstate-lock) is the standard pattern for
# preventing concurrent applies from corrupting state. Both were
# created manually before this module was first init'd — see the
# README for the one-time setup commands.
terraform {
  required_version = ">= 1.7.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.70"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }

  backend "s3" {
    bucket         = "havoc-tfstate-802531653822"
    key            = "havoc.tfstate"
    region         = "us-east-1"
    dynamodb_table = "havoc-tfstate-lock"
    encrypt        = true
  }
}
