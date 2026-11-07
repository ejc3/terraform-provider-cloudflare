resource "cloudflare_zero_trust_access_application" "%[1]s" {
  zone_id                       = "%[2]s"
  name                          = "%[1]s"
  type                          = "self_hosted"
  session_duration              = "24h"
  eager_redirect_cookie_setting = %[4]s
  destinations = [
    { type = "public", uri = "%[1]s.%[3]s" },
    { type = "public", uri = "%[1]s-api.%[3]s" },
  ]
}
