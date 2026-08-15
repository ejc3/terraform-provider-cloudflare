package workers_build_trigger_environment_variables

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/importpath"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigure = (*WorkersBuildTriggerEnvironmentVariablesResource)(nil)
var _ resource.ResourceWithImportState = (*WorkersBuildTriggerEnvironmentVariablesResource)(nil)

func NewResource() resource.Resource { return &WorkersBuildTriggerEnvironmentVariablesResource{} }

type WorkersBuildTriggerEnvironmentVariablesResource struct{ client *cloudflare.Client }

func (r *WorkersBuildTriggerEnvironmentVariablesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workers_build_trigger_environment_variables"
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*cloudflare.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected resource configure type", fmt.Sprintf("Expected *cloudflare.Client, got %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkersBuildTriggerEnvironmentVariablesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	variables, diags := variablesFromTerraform(ctx, data.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiVariables, err := r.upsert(ctx, data.AccountID.ValueString(), data.TriggerUUID.ValueString(), variables)
	if err != nil {
		resp.Diagnostics.AddError("failed to create Workers Builds environment variables", err.Error())
		return
	}
	if err := r.deleteUnexpected(ctx, data.AccountID.ValueString(), data.TriggerUUID.ValueString(), variables, apiVariables); err != nil {
		resp.Diagnostics.AddError("failed to delete undeclared Workers Builds environment variable", err.Error())
		return
	}
	data.ID = data.TriggerUUID
	resp.Diagnostics.Append(mergeDesiredAPIState(ctx, &data, variables, apiVariables)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkersBuildTriggerEnvironmentVariablesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	prior, diags := variablesFromTerraform(ctx, data.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiVariables, status, err := r.list(ctx, data.AccountID.ValueString(), data.TriggerUUID.ValueString())
	if status == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("failed to read Workers Builds environment variables", err.Error())
		return
	}
	data.ID = data.TriggerUUID
	resp.Diagnostics.Append(mergeAPIState(ctx, &data, prior, apiVariables)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state WorkersBuildTriggerEnvironmentVariablesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	desired, diags := variablesFromTerraform(ctx, plan.Variables)
	resp.Diagnostics.Append(diags...)
	previous, diags := variablesFromTerraform(ctx, state.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, name := range removedVariableNames(previous, desired) {
		if err := r.deleteOne(ctx, plan.AccountID.ValueString(), plan.TriggerUUID.ValueString(), name); err != nil {
			resp.Diagnostics.AddError("failed to delete removed Workers Builds environment variable", err.Error())
			return
		}
	}
	apiVariables, err := r.upsert(ctx, plan.AccountID.ValueString(), plan.TriggerUUID.ValueString(), desired)
	if err != nil {
		resp.Diagnostics.AddError("failed to update Workers Builds environment variables", err.Error())
		return
	}
	if err := r.deleteUnexpected(ctx, plan.AccountID.ValueString(), plan.TriggerUUID.ValueString(), desired, apiVariables); err != nil {
		resp.Diagnostics.AddError("failed to delete undeclared Workers Builds environment variable", err.Error())
		return
	}
	plan.ID = plan.TriggerUUID
	resp.Diagnostics.Append(mergeDesiredAPIState(ctx, &plan, desired, apiVariables)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkersBuildTriggerEnvironmentVariablesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	variables, diags := variablesFromTerraform(ctx, data.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	names := make([]string, 0, len(variables))
	for name := range variables {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := r.deleteOne(ctx, data.AccountID.ValueString(), data.TriggerUUID.ValueString(), name); err != nil {
			resp.Diagnostics.AddError("failed to delete Workers Builds environment variable", err.Error())
			return
		}
	}
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var accountID, triggerUUID string
	resp.Diagnostics.Append(importpath.ParseImportID(req.ID, "<account_id>/<trigger_uuid>", &accountID, &triggerUUID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiVariables, status, err := r.list(ctx, accountID, triggerUUID)
	if status == http.StatusNotFound {
		resp.Diagnostics.AddError("Workers Builds trigger not found", "The trigger does not exist.")
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("failed to import Workers Builds environment variables", err.Error())
		return
	}
	data := WorkersBuildTriggerEnvironmentVariablesModel{
		ID:          types.StringValue(triggerUUID),
		AccountID:   types.StringValue(accountID),
		TriggerUUID: types.StringValue(triggerUUID),
	}
	resp.Diagnostics.Append(mergeAPIState(ctx, &data, nil, apiVariables)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) list(ctx context.Context, accountID, triggerUUID string) (map[string]environmentVariableResponse, int, error) {
	var env environmentVariablesEnvelope
	status, err := r.execute(ctx, http.MethodGet, environmentVariablesPath(accountID, triggerUUID), nil, &env)
	if err == nil {
		err = validateEnvelope(env.Success, env.Errors)
	}
	return env.Result, status, err
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) upsert(ctx context.Context, accountID, triggerUUID string, variables map[string]EnvironmentVariableModel) (map[string]environmentVariableResponse, error) {
	body := make(map[string]environmentVariableRequest, len(variables))
	for name, variable := range variables {
		var value *string
		if !variable.Value.IsNull() && !variable.Value.IsUnknown() {
			plain := variable.Value.ValueString()
			value = &plain
		}
		body[name] = environmentVariableRequest{Value: value, IsSecret: variable.IsSecret.ValueBool()}
	}
	var env environmentVariablesEnvelope
	_, err := r.execute(ctx, http.MethodPatch, environmentVariablesPath(accountID, triggerUUID), body, &env)
	if err == nil {
		err = validateEnvelope(env.Success, env.Errors)
	}
	return env.Result, err
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) deleteOne(ctx context.Context, accountID, triggerUUID, name string) error {
	path := environmentVariablesPath(accountID, triggerUUID) + "/" + url.PathEscape(name)
	var env deleteEnvelope
	status, err := r.execute(ctx, http.MethodDelete, path, nil, &env)
	if status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	return validateEnvelope(env.Success, env.Errors)
}

func environmentVariablesPath(accountID, triggerUUID string) string {
	return fmt.Sprintf("accounts/%s/builds/triggers/%s/environment_variables", url.PathEscape(accountID), url.PathEscape(triggerUUID))
}

// execute intentionally omits the provider's generic request/response logging
// middleware because these payloads can contain build secrets.
func (r *WorkersBuildTriggerEnvironmentVariablesResource) execute(ctx context.Context, method, path string, body any, result any) (int, error) {
	options := make([]option.RequestOption, 0, 1)
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("serialize request: %w", err)
		}
		options = append(options, option.WithRequestBody("application/json", encoded))
	}
	response := new(http.Response)
	options = append(options, option.WithResponseInto(&response))
	err := r.client.Execute(ctx, method, path, nil, result, options...)
	if response == nil {
		if err != nil {
			return 0, fmt.Errorf("Cloudflare API request failed before a response was received: %T", err)
		}
		return 0, nil
	}
	if err != nil {
		if response.StatusCode != 0 {
			return response.StatusCode, fmt.Errorf("Cloudflare API request failed with HTTP status %d; the response body is omitted because this resource handles secrets", response.StatusCode)
		}
		return 0, fmt.Errorf("Cloudflare API request failed before a response was received: %T", err)
	}
	return response.StatusCode, nil
}

func validateEnvelope(success bool, errors []apiError) error {
	if success && len(errors) == 0 {
		return nil
	}
	if len(errors) == 0 {
		return fmt.Errorf("Cloudflare returned success=false")
	}
	// Error messages are deliberately omitted. Cloudflare can echo a rejected
	// environment-variable value in the message, including a secret submitted
	// in this request.
	return fmt.Errorf("Cloudflare returned error code %d; the response message is omitted because this resource handles secrets", errors[0].Code)
}

func variablesFromTerraform(ctx context.Context, value types.Map) (map[string]EnvironmentVariableModel, diag.Diagnostics) {
	result := map[string]EnvironmentVariableModel{}
	if value.IsNull() || value.IsUnknown() {
		return result, nil
	}
	diagnostics := value.ElementsAs(ctx, &result, false)
	return result, diagnostics
}

func removedVariableNames(previous, desired map[string]EnvironmentVariableModel) []string {
	var names []string
	for name := range previous {
		if _, exists := desired[name]; !exists {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func unexpectedRemoteVariableNames(api map[string]environmentVariableResponse, desired map[string]EnvironmentVariableModel) []string {
	var names []string
	for name := range api {
		if _, exists := desired[name]; !exists {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) deleteUnexpected(ctx context.Context, accountID, triggerUUID string, desired map[string]EnvironmentVariableModel, api map[string]environmentVariableResponse) error {
	for _, name := range unexpectedRemoteVariableNames(api, desired) {
		if err := r.deleteOne(ctx, accountID, triggerUUID, name); err != nil {
			return err
		}
	}
	return nil
}

func mergeAPIState(ctx context.Context, data *WorkersBuildTriggerEnvironmentVariablesModel, prior map[string]EnvironmentVariableModel, api map[string]environmentVariableResponse) diag.Diagnostics {
	variables := make(map[string]EnvironmentVariableModel, len(api))
	for name, remote := range api {
		previous, hasPrevious := prior[name]
		value := types.StringNull()
		if remote.Value != nil {
			value = types.StringValue(*remote.Value)
		} else if remote.IsSecret {
			if hasPrevious && !previous.Value.IsNull() && !previous.Value.IsUnknown() {
				value = previous.Value
			}
		}
		variables[name] = EnvironmentVariableModel{Value: value, IsSecret: types.BoolValue(remote.IsSecret)}
	}
	return setVariablesState(ctx, data, variables)
}

func mergeDesiredAPIState(ctx context.Context, data *WorkersBuildTriggerEnvironmentVariablesModel, desired map[string]EnvironmentVariableModel, api map[string]environmentVariableResponse) diag.Diagnostics {
	variables := make(map[string]EnvironmentVariableModel, len(desired))
	for name, configured := range desired {
		remote, exists := api[name]
		if !exists {
			variables[name] = configured
			continue
		}
		value := types.StringNull()
		if remote.Value != nil {
			value = types.StringValue(*remote.Value)
		} else if !configured.Value.IsNull() && !configured.Value.IsUnknown() {
			// PATCH responses can omit a variable value even when they include
			// the variable itself. Preserve the configured value so a partial
			// response cannot erase either a plain value or a write-only secret
			// from the post-apply state.
			value = configured.Value
		}
		variables[name] = EnvironmentVariableModel{Value: value, IsSecret: types.BoolValue(remote.IsSecret)}
	}
	return setVariablesState(ctx, data, variables)
}

func setVariablesState(ctx context.Context, data *WorkersBuildTriggerEnvironmentVariablesModel, variables map[string]EnvironmentVariableModel) diag.Diagnostics {
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"value": types.StringType, "is_secret": types.BoolType}}
	state, diagnostics := types.MapValueFrom(ctx, objectType, variables)
	data.Variables = state
	return diagnostics
}
