package workers_build_token

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/importpath"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const buildTokenListPageSize = 200

var _ resource.ResourceWithConfigure = (*WorkersBuildTokenResource)(nil)
var _ resource.ResourceWithImportState = (*WorkersBuildTokenResource)(nil)

func NewResource() resource.Resource {
	return &WorkersBuildTokenResource{}
}

// WorkersBuildTokenResource manages a user-owned API token registration used
// by Workers Builds to deploy Workers.
type WorkersBuildTokenResource struct {
	client *cloudflare.Client
}

func (r *WorkersBuildTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workers_build_token"
}

func (r *WorkersBuildTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*cloudflare.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("Expected *cloudflare.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *WorkersBuildTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkersBuildTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := buildTokenRequest{
		BuildTokenName:    data.BuildTokenName.ValueString(),
		BuildTokenSecret:  data.BuildTokenSecret.ValueString(),
		CloudflareTokenID: data.CloudflareTokenID.ValueString(),
	}
	path := fmt.Sprintf("accounts/%s/builds/tokens", url.PathEscape(data.AccountID.ValueString()))
	res := new(http.Response)

	// Do not attach the provider's normal request logging middleware here: it
	// logs JSON request bodies, and this body contains build_token_secret.
	err := r.client.Post(ctx, path, payload, &res)
	if err != nil {
		if res != nil && res.Body != nil {
			_ = res.Body.Close()
		}
		resp.Diagnostics.AddError("failed to create Workers Builds token", safeRequestError(res, err))
		return
	}
	defer res.Body.Close()

	var envelope buildTokenEnvelope
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		resp.Diagnostics.AddError("failed to decode Workers Builds token response", err.Error())
		return
	}
	if !envelope.Success || len(envelope.Errors) != 0 {
		resp.Diagnostics.AddError("failed to create Workers Builds token", "Cloudflare returned an unsuccessful response.")
		return
	}
	if envelope.Result.BuildTokenUUID == "" {
		resp.Diagnostics.AddError("failed to create Workers Builds token", "Cloudflare returned an empty build token UUID.")
		return
	}
	if !supportedOwnerType(envelope.Result.OwnerType) {
		status, cleanupErr := r.deleteRegistration(ctx, data.AccountID.ValueString(), envelope.Result.BuildTokenUUID)
		detail := "Cloudflare created a token registration that is not explicitly user-owned; Workers Builds does not support account-owned or unknown-owner deployment tokens."
		if cleanupErr != nil {
			detail += fmt.Sprintf(" Automatic cleanup of registration %s failed: %s", envelope.Result.BuildTokenUUID, safeStatusError(status, cleanupErr))
		} else {
			detail += " The invalid registration was deleted automatically."
		}
		resp.Diagnostics.AddError("unsupported Workers Builds token owner", detail)
		return
	}

	applyBuildTokenResult(&data, envelope.Result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkersBuildTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, found, status, err := r.find(ctx, data.AccountID.ValueString(), data.ID.ValueString())
	if err == nil && !found {
		resp.Diagnostics.AddWarning("Workers Builds token not found", "The build token registration was not found and will be removed from Terraform state.")
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("failed to read Workers Builds token", safeStatusError(status, err))
		return
	}
	if !supportedOwnerType(result.OwnerType) {
		resp.Diagnostics.AddError("unsupported Workers Builds token owner", "The build token registration is account-owned; Workers Builds currently supports only user-owned deployment tokens.")
		return
	}

	secret := data.BuildTokenSecret
	applyBuildTokenResult(&data, result)
	data.BuildTokenSecret = secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkersBuildTokenModel
	var state WorkersBuildTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.BuildTokenSecret.IsNull() && !plan.BuildTokenSecret.IsNull() && !plan.BuildTokenSecret.IsUnknown() && immutableTokenFieldsMatch(plan, state) {
		// Import cannot recover the write-only secret. The first configured value
		// is an operator assertion that it matches the already-registered token;
		// reconcile state only and make no remote API call.
		plan.ID = state.ID
		plan.OwnerType = state.OwnerType
		resp.Diagnostics.AddWarning(
			"Workers Builds token secret reconciled from configuration",
			"Cloudflare never returns this secret, so the provider recorded the configured value without changing or validating the remote token registration.",
		)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	resp.Diagnostics.AddError(
		"unexpected Workers Builds token update",
		"Workers Builds token inputs require replacement. Only the one-time reconciliation of a null secret after import is an in-place state operation.",
	)
}

func (r *WorkersBuildTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkersBuildTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, err := r.deleteRegistration(ctx, data.AccountID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to delete Workers Builds token", safeStatusError(status, err))
	}
}

func (r *WorkersBuildTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var accountID string
	var buildTokenUUID string
	resp.Diagnostics.Append(importpath.ParseImportID(
		req.ID,
		"<account_id>/<build_token_uuid>",
		&accountID,
		&buildTokenUUID,
	)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, found, status, err := r.find(ctx, accountID, buildTokenUUID)
	if err != nil {
		resp.Diagnostics.AddError("failed to import Workers Builds token", safeStatusError(status, err))
		return
	}
	if !found {
		resp.Diagnostics.AddError("Workers Builds token not found", "No build token registration with the requested UUID exists in the account.")
		return
	}
	if !supportedOwnerType(result.OwnerType) {
		resp.Diagnostics.AddError("unsupported Workers Builds token owner", "The requested build token registration is account-owned; Workers Builds currently supports only user-owned deployment tokens.")
		return
	}
	if result.BuildTokenName == "" || result.CloudflareTokenID == "" {
		resp.Diagnostics.AddError("incomplete Workers Builds token response", "Cloudflare did not return the token name and Cloudflare token ID required to import this registration safely.")
		return
	}

	data := WorkersBuildTokenModel{
		AccountID:        types.StringValue(accountID),
		BuildTokenSecret: types.StringNull(),
	}
	applyBuildTokenResult(&data, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.AddWarning(
		"Workers Builds token secret must be reconciled",
		"Cloudflare does not return build_token_secret. Add the exact original user API token value to configuration after import; the provider will record that first value in state without changing the remote registration and cannot verify that it is correct.",
	)
}

func (r *WorkersBuildTokenResource) find(ctx context.Context, accountID, buildTokenUUID string) (buildTokenResult, bool, int, error) {
	for page := 1; ; page++ {
		path := fmt.Sprintf(
			"accounts/%s/builds/tokens?page=%d&per_page=%d",
			url.PathEscape(accountID),
			page,
			buildTokenListPageSize,
		)
		res := new(http.Response)
		if err := r.client.Get(ctx, path, nil, &res); err != nil {
			status := 0
			if res != nil {
				status = res.StatusCode
				if res.Body != nil {
					_ = res.Body.Close()
				}
			}
			return buildTokenResult{}, false, status, err
		}

		var envelope buildTokenListEnvelope
		decodeErr := json.NewDecoder(res.Body).Decode(&envelope)
		closeErr := res.Body.Close()
		if decodeErr != nil {
			return buildTokenResult{}, false, res.StatusCode, decodeErr
		}
		if closeErr != nil {
			return buildTokenResult{}, false, res.StatusCode, closeErr
		}
		if !envelope.Success || len(envelope.Errors) != 0 {
			return buildTokenResult{}, false, res.StatusCode, fmt.Errorf("Cloudflare returned an unsuccessful response")
		}

		for _, token := range envelope.Result {
			if token.BuildTokenUUID == buildTokenUUID {
				return token, true, res.StatusCode, nil
			}
		}

		if envelope.ResultInfo.TotalPages > 0 {
			if page >= envelope.ResultInfo.TotalPages {
				return buildTokenResult{}, false, res.StatusCode, nil
			}
			continue
		}
		// result_info is optional. A full page can therefore be evidence that a
		// later page exists, while a short page is the safe terminal condition.
		if len(envelope.Result) < buildTokenListPageSize {
			return buildTokenResult{}, false, res.StatusCode, nil
		}
	}
}

func (r *WorkersBuildTokenResource) deleteRegistration(ctx context.Context, accountID, buildTokenUUID string) (int, error) {
	path := fmt.Sprintf(
		"accounts/%s/builds/tokens/%s",
		url.PathEscape(accountID),
		url.PathEscape(buildTokenUUID),
	)
	res := new(http.Response)
	err := r.client.Delete(ctx, path, nil, &res)
	status := 0
	if res != nil {
		status = res.StatusCode
	}
	if status == http.StatusNotFound {
		if res != nil && res.Body != nil {
			_ = res.Body.Close()
		}
		return status, nil
	}
	if err != nil {
		if res != nil && res.Body != nil {
			_ = res.Body.Close()
		}
		return status, err
	}
	if res == nil || res.Body == nil {
		return status, fmt.Errorf("Cloudflare returned no response body")
	}
	defer res.Body.Close()

	var envelope struct {
		Success bool                 `json:"success"`
		Errors  []buildTokenAPIError `json:"errors"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return status, err
	}
	if !envelope.Success || len(envelope.Errors) != 0 {
		return status, fmt.Errorf("Cloudflare returned an unsuccessful response")
	}
	return status, nil
}

func applyBuildTokenResult(data *WorkersBuildTokenModel, result buildTokenResult) {
	data.ID = types.StringValue(result.BuildTokenUUID)
	if result.BuildTokenName != "" {
		data.BuildTokenName = types.StringValue(result.BuildTokenName)
	}
	if result.CloudflareTokenID != "" {
		data.CloudflareTokenID = types.StringValue(result.CloudflareTokenID)
	}
	if result.OwnerType == "" {
		data.OwnerType = types.StringNull()
	} else {
		data.OwnerType = types.StringValue(result.OwnerType)
	}
}

func supportedOwnerType(ownerType string) bool {
	return ownerType == "user"
}

func immutableTokenFieldsMatch(plan, state WorkersBuildTokenModel) bool {
	return plan.AccountID.ValueString() == state.AccountID.ValueString() &&
		plan.BuildTokenName.ValueString() == state.BuildTokenName.ValueString() &&
		plan.CloudflareTokenID.ValueString() == state.CloudflareTokenID.ValueString()
}

func safeRequestError(res *http.Response, err error) string {
	status := 0
	if res != nil {
		status = res.StatusCode
	}
	return safeStatusError(status, err)
}

func safeStatusError(status int, err error) string {
	if status >= http.StatusBadRequest {
		return fmt.Sprintf("Cloudflare API request failed with HTTP status %d. The response body is omitted because this resource handles a secret.", status)
	}
	return fmt.Sprintf("Cloudflare API request or response could not be used: %T. Details are omitted because this resource handles a secret.", err)
}
