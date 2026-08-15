package workers_subdomain

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/cloudflare/cloudflare-go/v7/workers"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/importpath"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/logging"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigure = (*WorkersSubdomainResource)(nil)
var _ resource.ResourceWithImportState = (*WorkersSubdomainResource)(nil)

func NewResource() resource.Resource {
	return &WorkersSubdomainResource{}
}

// WorkersSubdomainResource manages the one workers.dev label assigned to a
// Cloudflare account. Per-script exposure remains owned by
// cloudflare_workers_script_subdomain.
type WorkersSubdomainResource struct {
	client *cloudflare.Client
}

func (r *WorkersSubdomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workers_subdomain"
}

func (r *WorkersSubdomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkersSubdomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkersSubdomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.put(ctx, data.AccountID.ValueString(), data.Subdomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to make http request", err.Error())
		return
	}

	setResourceResult(&data, result.Subdomain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersSubdomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkersSubdomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Workers.Subdomains.Get(
		ctx,
		workers.SubdomainGetParams{AccountID: cloudflare.F(data.AccountID.ValueString())},
		option.WithMiddleware(logging.Middleware(ctx)),
	)
	if isNotFound(err) {
		resp.Diagnostics.AddWarning("Resource not found", "The Workers subdomain was not found and will be removed from state.")
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("failed to make http request", err.Error())
		return
	}

	setResourceResult(&data, result.Subdomain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersSubdomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkersSubdomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.put(ctx, data.AccountID.ValueString(), data.Subdomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to make http request", err.Error())
		return
	}

	setResourceResult(&data, result.Subdomain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersSubdomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkersSubdomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Workers.Subdomains.Delete(
		ctx,
		workers.SubdomainDeleteParams{AccountID: cloudflare.F(data.AccountID.ValueString())},
		option.WithMiddleware(logging.Middleware(ctx)),
	)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("failed to make http request", err.Error())
	}
}

func (r *WorkersSubdomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	accountID := ""
	resp.Diagnostics.Append(importpath.ParseImportID(req.ID, "<account_id>", &accountID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if accountID == "" {
		resp.Diagnostics.AddError("invalid ID", "account ID must not be empty")
		return
	}

	result, err := r.client.Workers.Subdomains.Get(
		ctx,
		workers.SubdomainGetParams{AccountID: cloudflare.F(accountID)},
		option.WithMiddleware(logging.Middleware(ctx)),
	)
	if err != nil {
		resp.Diagnostics.AddError("failed to make http request", err.Error())
		return
	}

	data := WorkersSubdomainModel{
		ID:        types.StringValue(accountID),
		AccountID: types.StringValue(accountID),
		Subdomain: types.StringValue(result.Subdomain),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersSubdomainResource) put(ctx context.Context, accountID, subdomain string) (*workers.SubdomainUpdateResponse, error) {
	return r.client.Workers.Subdomains.Update(
		ctx,
		workers.SubdomainUpdateParams{
			AccountID: cloudflare.F(accountID),
			Subdomain: cloudflare.F(subdomain),
		},
		option.WithMiddleware(logging.Middleware(ctx)),
	)
}

func setResourceResult(data *WorkersSubdomainModel, subdomain string) {
	data.ID = data.AccountID
	data.Subdomain = types.StringValue(subdomain)
}

func isNotFound(err error) bool {
	var apiErr *cloudflare.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
