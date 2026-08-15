resource "cloudflare_workers_build_repository_connection" "colton_games" {
  account_id            = "12ea67fb7ced068de03f35c22688e436"
  provider_type         = "github"
  provider_account_id   = "250920182"
  provider_account_name = "CoderColton"
  repo_id               = "1120877379"
  repo_name             = "colton-games"
}
