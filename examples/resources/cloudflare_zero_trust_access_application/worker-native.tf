resource "cloudflare_zero_trust_access_application" "example_worker_native_access" {
  account_id       = "account_id"
  name             = "Worker-native Access"
  type             = "self_hosted"
  session_duration = "24h"

  # A Worker destination protects the Worker on every route, Custom Domain,
  # workers.dev hostname, and preview deployment. No domain is required.
  destinations = [{
    type      = "worker"
    worker_id = "72edf31f83e240448fce38bef56104e3"
  }]

  policies = [{
    id         = "f174e90a-fafe-4643-bbbc-4a0ed4fc8415"
    precedence = 1
  }]
}
