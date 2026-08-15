package workers_build_token

import "github.com/hashicorp/terraform-plugin-framework/types"

// WorkersBuildTokenModel is the Terraform state model for a Workers Builds
// deployment token registration.
type WorkersBuildTokenModel struct {
	ID                types.String `tfsdk:"id"`
	AccountID         types.String `tfsdk:"account_id"`
	BuildTokenName    types.String `tfsdk:"build_token_name"`
	BuildTokenSecret  types.String `tfsdk:"build_token_secret"`
	CloudflareTokenID types.String `tfsdk:"cloudflare_token_id"`
	OwnerType         types.String `tfsdk:"owner_type"`
}

type buildTokenRequest struct {
	BuildTokenName    string `json:"build_token_name"`
	BuildTokenSecret  string `json:"build_token_secret"`
	CloudflareTokenID string `json:"cloudflare_token_id"`
}

type buildTokenResult struct {
	BuildTokenName    string `json:"build_token_name"`
	BuildTokenUUID    string `json:"build_token_uuid"`
	CloudflareTokenID string `json:"cloudflare_token_id"`
	OwnerType         string `json:"owner_type"`
}

type buildTokenEnvelope struct {
	Result  buildTokenResult     `json:"result"`
	Success bool                 `json:"success"`
	Errors  []buildTokenAPIError `json:"errors"`
}

type buildTokenListEnvelope struct {
	Result     []buildTokenResult   `json:"result"`
	Success    bool                 `json:"success"`
	Errors     []buildTokenAPIError `json:"errors"`
	ResultInfo struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

type buildTokenAPIError struct {
	Code int `json:"code"`
}
