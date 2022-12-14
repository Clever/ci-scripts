variable "SHORT_SHA" {}

variable "ECR_ACCOUNT_ID" {}

variable "REPO" {}

variable "CACHE_SHA" {}

target "image" {
  dockerfile = "Dockerfile"
  tags = [
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-west-1.amazonaws.com/${REPO}:${SHORT_SHA}",
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-west-2.amazonaws.com/${REPO}:${SHORT_SHA}",
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/${REPO}:${SHORT_SHA}",
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-east-2.amazonaws.com/${REPO}:${SHORT_SHA}",
  ]
  cache-from = ["type=registry,ref=${ECR_ACCOUNT_ID}.dkr.ecr.us-west-1.amazonaws.com/${REPO}${CACHE_SHA}"]
  cache-to = ["type=inline"]
}
