terraform {
  required_providers {
    fortresswaf = {
      source = "fortresswaf/fortresswaf"
      version = "~> 1.0"
    }
  }
}

provider "fortresswaf" {
  endpoint = var.fortress_endpoint
  api_key  = var.fortress_api_key
}
