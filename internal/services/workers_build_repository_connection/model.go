package workers_build_repository_connection

import "github.com/hashicorp/terraform-plugin-framework/types"

// WorkersBuildRepositoryConnectionModel is the Terraform state model for a
// Workers Builds repository connection.
type WorkersBuildRepositoryConnectionModel struct {
	ID                  types.String `tfsdk:"id"`
	AccountID           types.String `tfsdk:"account_id"`
	ProviderType        types.String `tfsdk:"provider_type"`
	ProviderAccountID   types.String `tfsdk:"provider_account_id"`
	ProviderAccountName types.String `tfsdk:"provider_account_name"`
	RepoID              types.String `tfsdk:"repo_id"`
	RepoName            types.String `tfsdk:"repo_name"`
}

type repositoryConnectionRequest struct {
	ProviderType        string `json:"provider_type"`
	ProviderAccountID   string `json:"provider_account_id"`
	ProviderAccountName string `json:"provider_account_name"`
	RepoID              string `json:"repo_id"`
	RepoName            string `json:"repo_name"`
}

type repositoryConnectionResult struct {
	RepositoryConnectionUUID string `json:"repo_connection_uuid"`
	ProviderType             string `json:"provider_type"`
	ProviderAccountID        string `json:"provider_account_id"`
	ProviderAccountName      string `json:"provider_account_name"`
	RepoID                   string `json:"repo_id"`
	RepoName                 string `json:"repo_name"`
}

type repositoryConnectionEnvelope struct {
	Result  repositoryConnectionResult     `json:"result"`
	Success bool                           `json:"success"`
	Errors  []repositoryConnectionAPIError `json:"errors"`
}

type repositoryConnectionAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
