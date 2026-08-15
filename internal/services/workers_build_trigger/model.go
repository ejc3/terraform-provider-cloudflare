package workers_build_trigger

import "github.com/hashicorp/terraform-plugin-framework/types"

// WorkersBuildTriggerModel is the Terraform state model for a Workers Builds
// production or preview trigger.
type WorkersBuildTriggerModel struct {
	ID                   types.String `tfsdk:"id"`
	AccountID            types.String `tfsdk:"account_id"`
	ExternalScriptID     types.String `tfsdk:"external_script_id"`
	RepositoryConnection types.String `tfsdk:"repository_connection_uuid"`
	BuildTokenUUID       types.String `tfsdk:"build_token_uuid"`
	TriggerName          types.String `tfsdk:"trigger_name"`
	BuildCommand         types.String `tfsdk:"build_command"`
	DeployCommand        types.String `tfsdk:"deploy_command"`
	RootDirectory        types.String `tfsdk:"root_directory"`
	BranchIncludes       types.Set    `tfsdk:"branch_includes"`
	BranchExcludes       types.Set    `tfsdk:"branch_excludes"`
	PathIncludes         types.Set    `tfsdk:"path_includes"`
	PathExcludes         types.Set    `tfsdk:"path_excludes"`
	BuildCachingEnabled  types.Bool   `tfsdk:"build_caching_enabled"`
	CreatedOn            types.String `tfsdk:"created_on"`
	ModifiedOn           types.String `tfsdk:"modified_on"`
}

type triggerRequest struct {
	ExternalScriptID     string   `json:"external_script_id,omitempty"`
	RepositoryConnection string   `json:"repo_connection_uuid,omitempty"`
	BuildTokenUUID       string   `json:"build_token_uuid,omitempty"`
	TriggerName          string   `json:"trigger_name"`
	BuildCommand         string   `json:"build_command"`
	DeployCommand        string   `json:"deploy_command"`
	RootDirectory        string   `json:"root_directory"`
	BranchIncludes       []string `json:"branch_includes"`
	BranchExcludes       []string `json:"branch_excludes"`
	PathIncludes         []string `json:"path_includes"`
	PathExcludes         []string `json:"path_excludes"`
	BuildCachingEnabled  bool     `json:"build_caching_enabled"`
}

type triggerAPIModel struct {
	TriggerUUID         *string   `json:"trigger_uuid"`
	ExternalScriptID    *string   `json:"external_script_id"`
	BuildTokenUUID      *string   `json:"build_token_uuid"`
	TriggerName         *string   `json:"trigger_name"`
	BuildCommand        *string   `json:"build_command"`
	DeployCommand       *string   `json:"deploy_command"`
	RootDirectory       *string   `json:"root_directory"`
	BranchIncludes      *[]string `json:"branch_includes"`
	BranchExcludes      *[]string `json:"branch_excludes"`
	PathIncludes        *[]string `json:"path_includes"`
	PathExcludes        *[]string `json:"path_excludes"`
	BuildCachingEnabled *bool     `json:"build_caching_enabled"`
	CreatedOn           *string   `json:"created_on"`
	ModifiedOn          *string   `json:"modified_on"`
	DeletedOn           *string   `json:"deleted_on"`
	Repository          *struct {
		UUID *string `json:"repo_connection_uuid"`
	} `json:"repo_connection"`
}

type triggerEnvelope struct {
	Result  triggerAPIModel `json:"result"`
	Success bool            `json:"success"`
	Errors  []apiError      `json:"errors"`
}

type triggerListEnvelope struct {
	Result     []triggerAPIModel `json:"result"`
	Success    bool              `json:"success"`
	Errors     []apiError        `json:"errors"`
	ResultInfo struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

type deleteEnvelope struct {
	Success bool       `json:"success"`
	Errors  []apiError `json:"errors"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
