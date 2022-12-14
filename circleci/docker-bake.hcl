variable "SHORT_SHA" {
  default = "latest"
}

target "image" {
  dockerfile = "Dockerfile"
  tags = [
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-west-1.amazonaws.com/$REPO:$SHORT_SHA",
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-west-2.amazonaws.com/$REPO:$SHORT_SHA",
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/$REPO:$SHORT_SHA",
    "${ECR_ACCOUNT_ID}.dkr.ecr.us-east-2.amazonaws.com/$REPO:$SHORT_SHA",
  ]
  cache-from = ["type=gha,scope=${REPO}"]
  cache-to = ["type=gha,mode=max,scope=${REPO}"]
}
