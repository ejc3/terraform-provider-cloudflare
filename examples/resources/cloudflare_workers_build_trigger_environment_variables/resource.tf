resource "cloudflare_workers_build_trigger_environment_variables" "preview" {
  account_id   = "023e105f4ecef8ad9ca31a8372d0c353"
  trigger_uuid = cloudflare_workers_build_trigger.preview.id

  variables = {
    CLOUDFLARE_ACCOUNT_ID = {
      value     = "023e105f4ecef8ad9ca31a8372d0c353"
      is_secret = false
    }
    WRANGLER_CI_GENERATE_PREVIEW_ALIAS = {
      value     = "true"
      is_secret = false
    }
  }
}
