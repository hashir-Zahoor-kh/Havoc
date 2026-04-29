# AWS provider config. Region is variabilized; auth is whatever
# `aws configure` left in ~/.aws/credentials (no SP/role chain yet —
# that's a Phase 6c concern when CI gets wired up).
#
# default_tags are applied to every resource the provider creates,
# so we don't have to repeat `tags = var.tags` on each resource
# block. Resources that already set a `tags` attribute will merge.
provider "aws" {
  region = var.region

  default_tags {
    tags = var.tags
  }
}
