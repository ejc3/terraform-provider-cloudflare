package workers_build_repository_connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/logging"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigure = (*WorkersBuildRepositoryConnectionResource)(nil)
var _ resource.ResourceWithImportState = (*WorkersBuildRepositoryConnectionResource)(nil)

func NewResource() resource.Resource {
	return &WorkersBuildRepositoryConnectionResource{}
}

// WorkersBuildRepositoryConnectionResource manages a connection between a Git
// repository and Workers Builds.
type WorkersBuildRepositoryConnectionResource struct {
	client *cloudflare.Client
}

func (r *WorkersBuildRepositoryConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workers_build_repository_connection"
}

func (r *WorkersBuildRepositoryConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkersBuildRepositoryConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkersBuildRepositoryConnectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := repositoryConnectionRequest{
		ProviderType:        data.ProviderType.ValueString(),
		ProviderAccountID:   data.ProviderAccountID.ValueString(),
		ProviderAccountName: data.ProviderAccountName.ValueString(),
		RepoID:              data.RepoID.ValueString(),
		RepoName:            data.RepoName.ValueString(),
	}

	path := fmt.Sprintf("accounts/%s/builds/repos/connections", url.PathEscape(data.AccountID.ValueString()))
	res := new(http.Response)
	err := r.client.Put(ctx, path, payload, &res, option.WithMiddleware(logging.Middleware(ctx)))
	if err != nil {
		if res != nil && res.Body != nil {
			_ = res.Body.Close()
		}
		resp.Diagnostics.AddError("failed to create Workers Builds repository connection", repositoryConnectionCreateError(data, err.Error()))
		return
	}
	defer res.Body.Close()

	var envelope repositoryConnectionEnvelope
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		resp.Diagnostics.AddError("failed to decode Workers Builds repository connection response", err.Error())
		return
	}
	if !envelope.Success || len(envelope.Errors) != 0 {
		resp.Diagnostics.AddError("failed to create Workers Builds repository connection", repositoryConnectionCreateError(data, "Cloudflare returned an unsuccessful response."))
		return
	}
	if envelope.Result.RepositoryConnectionUUID == "" {
		resp.Diagnostics.AddError("failed to create Workers Builds repository connection", "Cloudflare returned an empty repository connection UUID.")
		return
	}

	data.ID = types.StringValue(envelope.Result.RepositoryConnectionUUID)
	applyRepositoryConnectionResult(&data, envelope.Result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildRepositoryConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkersBuildRepositoryConnectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cloudflare exposes no GET or list endpoint for repository connections.
	// Preserve the last confirmed state rather than guessing that the remote
	// object was deleted and silently scheduling a replacement.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkersBuildRepositoryConnectionResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"unexpected Workers Builds repository connection update",
		"All repository connection inputs require replacement; report this provider error if Terraform attempted an in-place update.",
	)
}

func (r *WorkersBuildRepositoryConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkersBuildRepositoryConnectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := fmt.Sprintf(
		"accounts/%s/builds/repos/connections/%s",
		url.PathEscape(data.AccountID.ValueString()),
		url.PathEscape(data.ID.ValueString()),
	)
	res := new(http.Response)
	err := r.client.Delete(ctx, path, nil, &res, option.WithMiddleware(logging.Middleware(ctx)))
	if res != nil && res.StatusCode == http.StatusNotFound {
		if res.Body != nil {
			_ = res.Body.Close()
		}
		return
	}
	if err != nil {
		if res != nil && res.Body != nil {
			_ = res.Body.Close()
		}
		resp.Diagnostics.AddError("failed to delete Workers Builds repository connection", err.Error())
		return
	}
	defer res.Body.Close()

	var envelope struct {
		Success bool                           `json:"success"`
		Errors  []repositoryConnectionAPIError `json:"errors"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		resp.Diagnostics.AddError("failed to decode Workers Builds repository connection delete response", err.Error())
		return
	}
	if !envelope.Success || len(envelope.Errors) != 0 {
		resp.Diagnostics.AddError("failed to delete Workers Builds repository connection", "Cloudflare returned an unsuccessful response.")
	}
}

func (r *WorkersBuildRepositoryConnectionResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Workers Builds repository connections cannot be imported safely",
		"Cloudflare exposes no GET or list endpoint for repository connections, so the provider cannot verify an imported UUID or hydrate its required attributes. Declare a new connection in configuration instead; existing out-of-band connections must remain outside Terraform state.",
	)
}

func repositoryConnectionCreateError(data WorkersBuildRepositoryConnectionModel, apiContext string) string {
	if data.ProviderType.ValueString() != "github" {
		return apiContext
	}
	return fmt.Sprintf(
		"%s Ensure the Cloudflare Workers and Pages GitHub App is installed and authorized for the private repository %s/%s, then retry.",
		apiContext,
		data.ProviderAccountName.ValueString(),
		data.RepoName.ValueString(),
	)
}

func applyRepositoryConnectionResult(data *WorkersBuildRepositoryConnectionModel, result repositoryConnectionResult) {
	if result.ProviderType != "" {
		data.ProviderType = types.StringValue(result.ProviderType)
	}
	if result.ProviderAccountID != "" {
		data.ProviderAccountID = types.StringValue(result.ProviderAccountID)
	}
	if result.ProviderAccountName != "" {
		data.ProviderAccountName = types.StringValue(result.ProviderAccountName)
	}
	if result.RepoID != "" {
		data.RepoID = types.StringValue(result.RepoID)
	}
	if result.RepoName != "" {
		data.RepoName = types.StringValue(result.RepoName)
	}
}
