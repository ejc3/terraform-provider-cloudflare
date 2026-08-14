resource "cloudflare_zero_trust_access_application" "%[1]s" {
  account_id = "%[2]s"
  name       = "%[1]s"
  type       = "self_hosted"

  destinations = [{
    type      = "preview_worker"
    worker_id = "%[3]s"
  }]
}
