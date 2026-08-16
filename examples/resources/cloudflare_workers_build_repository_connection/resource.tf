resource "cloudflare_workers_build_repository_connection" "example" {
  account_id            = "023e105f4ecef8ad9ca31a8372d0c353"
  provider_type         = "github"
  provider_account_id   = "123456789"
  provider_account_name = "example-org"
  repo_id               = "987654321"
  repo_name             = "example-worker"
}
