package workers_build_trigger_environment_variables

import "github.com/hashicorp/terraform-plugin-framework/types"

type WorkersBuildTriggerEnvironmentVariablesModel struct {
	ID          types.String `tfsdk:"id"`
	AccountID   types.String `tfsdk:"account_id"`
	TriggerUUID types.String `tfsdk:"trigger_uuid"`
	Variables   types.Map    `tfsdk:"variables"`
}

type EnvironmentVariableModel struct {
	Value    types.String `tfsdk:"value"`
	IsSecret types.Bool   `tfsdk:"is_secret"`
}

type environmentVariableRequest struct {
	Value    *string `json:"value,omitempty"`
	IsSecret bool    `json:"is_secret"`
}

type environmentVariableResponse struct {
	Value     *string `json:"value"`
	IsSecret  bool    `json:"is_secret"`
	CreatedOn string  `json:"created_on"`
}

type environmentVariablesEnvelope struct {
	Result  map[string]environmentVariableResponse `json:"result"`
	Success bool                                   `json:"success"`
	Errors  []apiError                             `json:"errors"`
}

type deleteEnvelope struct {
	Success bool       `json:"success"`
	Errors  []apiError `json:"errors"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
