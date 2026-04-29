# One ECR repo per Havoc binary. CI pushes images here in Phase 6c;
# EKS nodes pull via the AmazonEC2ContainerRegistryReadOnly policy
# attached to the node role.
#
# image_tag_mutability = MUTABLE so `:dev` can be reused while
# iterating. For prod we'd flip to IMMUTABLE and require unique
# tags per build.
#
# A 10-image lifecycle keeps the bill at "essentially $0" — three
# small Go binaries × 10 images × ~30 MB ≈ 1 GB total at $0.10/GB.

resource "aws_ecr_repository" "this" {
  for_each             = toset(var.ecr_repositories)
  name                 = each.value
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "this" {
  for_each   = aws_ecr_repository.this
  repository = each.value.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep only the 10 most recent images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 10
      }
      action = { type = "expire" }
    }]
  })
}
