variable "workers_build_deployment_token" {
  type      = string
  sensitive = true
}

variable "workers_build_deployment_token_id" {
  type = string
}

resource "cloudflare_workers_build_token" "deployment" {
  account_id          = "12ea67fb7ced068de03f35c22688e436"
  build_token_name    = "terraform-deployment-token"
  build_token_secret  = var.workers_build_deployment_token
  cloudflare_token_id = var.workers_build_deployment_token_id
}
