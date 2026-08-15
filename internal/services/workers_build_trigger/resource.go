package workers_build_trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/importpath"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/logging"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigure = (*WorkersBuildTriggerResource)(nil)
var _ resource.ResourceWithImportState = (*WorkersBuildTriggerResource)(nil)

const triggerListMaxPages = 100

func NewResource() resource.Resource { return &WorkersBuildTriggerResource{} }

type WorkersBuildTriggerResource struct{ client *cloudflare.Client }

func (r *WorkersBuildTriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workers_build_trigger"
}

func (r *WorkersBuildTriggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkersBuildTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkersBuildTriggerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, diags := requestFromModel(ctx, data, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var env triggerEnvelope
	if _, err := r.execute(ctx, http.MethodPost, fmt.Sprintf("accounts/%s/builds/triggers", url.PathEscape(data.AccountID.ValueString())), body, &env); err != nil {
		resp.Diagnostics.AddError("failed to create Workers Builds trigger", err.Error())
		return
	}
	if err := validateEnvelope(env.Success, env.Errors); err != nil {
		resp.Diagnostics.AddError("Cloudflare rejected Workers Builds trigger creation", err.Error())
		return
	}
	if env.Result.TriggerUUID == nil || *env.Result.TriggerUUID == "" {
		resp.Diagnostics.AddError("failed to create Workers Builds trigger", "Cloudflare returned an empty trigger UUID.")
		return
	}
	resp.Diagnostics.Append(updateModelFromAPI(ctx, &data, env.Result)...)
	nullUnknownComputed(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkersBuildTriggerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, apiModel, err := r.find(ctx, data.AccountID.ValueString(), data.ExternalScriptID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to read Workers Builds trigger", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(updateModelFromAPI(ctx, &data, apiModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior WorkersBuildTriggerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mergeUnknownComputed(&data, prior)
	body, diags := requestFromModel(ctx, data, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var env triggerEnvelope
	path := fmt.Sprintf("accounts/%s/builds/triggers/%s", url.PathEscape(data.AccountID.ValueString()), url.PathEscape(data.ID.ValueString()))
	if _, err := r.execute(ctx, http.MethodPatch, path, body, &env); err != nil {
		resp.Diagnostics.AddError("failed to update Workers Builds trigger", err.Error())
		return
	}
	if err := validateEnvelope(env.Success, env.Errors); err != nil {
		resp.Diagnostics.AddError("Cloudflare rejected Workers Builds trigger update", err.Error())
		return
	}
	resp.Diagnostics.Append(updateModelFromAPI(ctx, &data, env.Result)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func nullUnknownComputed(data *WorkersBuildTriggerModel) {
	if data.CreatedOn.IsUnknown() {
		data.CreatedOn = types.StringNull()
	}
	if data.ModifiedOn.IsUnknown() {
		data.ModifiedOn = types.StringNull()
	}
}

func mergeUnknownComputed(data *WorkersBuildTriggerModel, prior WorkersBuildTriggerModel) {
	if data.ID.IsUnknown() {
		data.ID = prior.ID
	}
	if data.CreatedOn.IsUnknown() {
		data.CreatedOn = prior.CreatedOn
	}
	if data.ModifiedOn.IsUnknown() {
		data.ModifiedOn = prior.ModifiedOn
	}
}

func (r *WorkersBuildTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkersBuildTriggerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	path := fmt.Sprintf("accounts/%s/builds/triggers/%s", url.PathEscape(data.AccountID.ValueString()), url.PathEscape(data.ID.ValueString()))
	var env deleteEnvelope
	status, err := r.execute(ctx, http.MethodDelete, path, nil, &env)
	if err != nil && status != http.StatusNotFound {
		resp.Diagnostics.AddError("failed to delete Workers Builds trigger", err.Error())
		return
	}
	if status != http.StatusNotFound {
		if err := validateEnvelope(env.Success, env.Errors); err != nil {
			resp.Diagnostics.AddError("Cloudflare rejected Workers Builds trigger deletion", err.Error())
		}
	}
}

func (r *WorkersBuildTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var accountID, externalScriptID, triggerUUID string
	resp.Diagnostics.Append(importpath.ParseImportID(req.ID, "<account_id>/<external_script_id>/<trigger_uuid>", &accountID, &externalScriptID, &triggerUUID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, apiModel, err := r.find(ctx, accountID, externalScriptID, triggerUUID)
	if err != nil {
		resp.Diagnostics.AddError("failed to import Workers Builds trigger", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Workers Builds trigger not found", "No trigger with the requested UUID exists for the Worker tag.")
		return
	}
	data := WorkersBuildTriggerModel{AccountID: types.StringValue(accountID), ExternalScriptID: types.StringValue(externalScriptID), ID: types.StringValue(triggerUUID)}
	resp.Diagnostics.Append(updateModelFromAPI(ctx, &data, apiModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTriggerResource) find(ctx context.Context, accountID, externalScriptID, triggerUUID string) (bool, triggerAPIModel, error) {
	basePath := fmt.Sprintf("accounts/%s/builds/workers/%s/triggers", url.PathEscape(accountID), url.PathEscape(externalScriptID))
	for page := 1; page <= triggerListMaxPages; page++ {
		var env triggerListEnvelope
		path := fmt.Sprintf("%s?page=%d", basePath, page)
		status, err := r.execute(ctx, http.MethodGet, path, nil, &env)
		if status == http.StatusNotFound {
			return false, triggerAPIModel{}, nil
		}
		if err != nil {
			return false, triggerAPIModel{}, err
		}
		if err := validateEnvelope(env.Success, env.Errors); err != nil {
			return false, triggerAPIModel{}, err
		}
		for _, trigger := range env.Result {
			if trigger.TriggerUUID != nil && *trigger.TriggerUUID == triggerUUID && (trigger.DeletedOn == nil || *trigger.DeletedOn == "") {
				return true, trigger, nil
			}
		}
		if env.ResultInfo == nil || env.ResultInfo.TotalPages <= 0 {
			return false, triggerAPIModel{}, fmt.Errorf("Cloudflare omitted pagination metadata before the managed Workers Builds trigger was found")
		}
		if page >= env.ResultInfo.TotalPages {
			return false, triggerAPIModel{}, nil
		}
	}
	return false, triggerAPIModel{}, fmt.Errorf("Cloudflare returned more Workers Builds trigger pages than the provider will traverse")
}

func (r *WorkersBuildTriggerResource) execute(ctx context.Context, method, path string, body any, result any) (int, error) {
	options := []option.RequestOption{option.WithMiddleware(logging.Middleware(ctx))}
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
		return 0, err
	}
	return response.StatusCode, err
}

func validateEnvelope(success bool, errors []apiError) error {
	if success && len(errors) == 0 {
		return nil
	}
	if len(errors) == 0 {
		return fmt.Errorf("Cloudflare returned success=false")
	}
	return fmt.Errorf("Cloudflare error %d: %s", errors[0].Code, errors[0].Message)
}

func requestFromModel(ctx context.Context, data WorkersBuildTriggerModel, includeImmutable bool) (triggerRequest, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	toStrings := func(set types.Set) []string {
		var values []string
		diagnostics.Append(set.ElementsAs(ctx, &values, false)...)
		slices.Sort(values)
		return values
	}
	result := triggerRequest{
		BuildTokenUUID:      data.BuildTokenUUID.ValueString(),
		TriggerName:         data.TriggerName.ValueString(),
		BuildCommand:        data.BuildCommand.ValueString(),
		DeployCommand:       data.DeployCommand.ValueString(),
		RootDirectory:       data.RootDirectory.ValueString(),
		BranchIncludes:      toStrings(data.BranchIncludes),
		BranchExcludes:      toStrings(data.BranchExcludes),
		PathIncludes:        toStrings(data.PathIncludes),
		PathExcludes:        toStrings(data.PathExcludes),
		BuildCachingEnabled: data.BuildCachingEnabled.ValueBool(),
	}
	if includeImmutable {
		result.ExternalScriptID = data.ExternalScriptID.ValueString()
		result.RepositoryConnection = data.RepositoryConnection.ValueString()
	}
	return result, diagnostics
}

func updateModelFromAPI(ctx context.Context, data *WorkersBuildTriggerModel, api triggerAPIModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if api.TriggerUUID != nil && *api.TriggerUUID != "" {
		data.ID = types.StringValue(*api.TriggerUUID)
	}
	if api.ExternalScriptID != nil {
		data.ExternalScriptID = types.StringValue(*api.ExternalScriptID)
	}
	if api.Repository != nil && api.Repository.UUID != nil && *api.Repository.UUID != "" {
		data.RepositoryConnection = types.StringValue(*api.Repository.UUID)
	}
	if api.BuildTokenUUID != nil {
		data.BuildTokenUUID = types.StringValue(*api.BuildTokenUUID)
	}
	if api.TriggerName != nil {
		data.TriggerName = types.StringValue(*api.TriggerName)
	}
	if api.BuildCommand != nil {
		data.BuildCommand = types.StringValue(*api.BuildCommand)
	}
	if api.DeployCommand != nil {
		data.DeployCommand = types.StringValue(*api.DeployCommand)
	}
	if api.RootDirectory != nil {
		data.RootDirectory = types.StringValue(*api.RootDirectory)
	}
	if api.BuildCachingEnabled != nil {
		data.BuildCachingEnabled = types.BoolValue(*api.BuildCachingEnabled)
	}
	if api.CreatedOn != nil {
		data.CreatedOn = types.StringValue(*api.CreatedOn)
	}
	if api.ModifiedOn != nil {
		data.ModifiedOn = types.StringValue(*api.ModifiedOn)
	}
	set := func(values []string) types.Set {
		value, diags := types.SetValueFrom(ctx, types.StringType, values)
		diagnostics.Append(diags...)
		return value
	}
	if api.BranchIncludes != nil {
		data.BranchIncludes = set(*api.BranchIncludes)
	}
	if api.BranchExcludes != nil {
		data.BranchExcludes = set(*api.BranchExcludes)
	}
	if api.PathIncludes != nil {
		data.PathIncludes = set(*api.PathIncludes)
	}
	if api.PathExcludes != nil {
		data.PathExcludes = set(*api.PathExcludes)
	}
	return diagnostics
}
