resource "cloudflare_workers_build_trigger" "preview" {
  account_id                 = "023e105f4ecef8ad9ca31a8372d0c353"
  external_script_id         = "72edf31f83e240448fce38bef56104e3"
  repository_connection_uuid = "11111111-1111-4111-8111-111111111111"
  build_token_uuid           = "22222222-2222-4222-8222-222222222222"
  trigger_name               = "Preview branches"
  build_command              = "npm run cf:build"
  deploy_command             = "npm run cf:upload:built"
  root_directory             = "/"
  branch_includes            = ["*"]
  branch_excludes            = ["main"]
  path_includes              = ["*"]
  path_excludes              = []
  build_caching_enabled      = true
}
