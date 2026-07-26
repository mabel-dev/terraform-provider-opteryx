terraform {
  required_providers {
    opteryx = {
      source  = "mabel-dev/opteryx"
      version = "~> 0.1"
    }
  }
}

# endpoint defaults to https://policy.opteryx.app (or OPTERYX_ENDPOINT).
# token is required and should come from OPTERYX_TOKEN, not be hardcoded here --
# it must be a bearer JWT for an owner/admin identity, and that identity cannot
# be a principal in any policy this provider manages (policy.opteryx rejects
# self-grants and self-modification).
provider "opteryx" {}
