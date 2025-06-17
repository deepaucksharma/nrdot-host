terraform {
  backend "s3" {
    bucket         = "platform-team-terraform-state"
    key            = "clean-platform/staging/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "platform-team-terraform-locks"
    
    # Use AWS IAM role for authentication
    # Configured via Grand Central terraform deployments
  }
}