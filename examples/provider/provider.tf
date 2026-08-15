terraform {
  required_providers {
    cloudflare = {
      source  = "ejc3/cloudflare"
      version = "= 5.24.0"
    }
  }
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

# Create a DNS record
resource "cloudflare_dns_record" "www" {
  # ...
}
