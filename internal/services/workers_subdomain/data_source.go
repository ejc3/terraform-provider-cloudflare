package workers_subdomain

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/cloudflare/cloudflare-go/v7/workers"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/logging"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*WorkersSubdomainDataSource)(nil)

func NewWorkersSubdomainDataSource() datasource.DataSource {
	return &WorkersSubdomainDataSource{}
}

type WorkersSubdomainDataSource struct {
	client *cloudflare.Client
}

func (d *WorkersSubdomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workers_subdomain"
}

func (d *WorkersSubdomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*cloudflare.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected data source configure type",
			fmt.Sprintf("Expected *cloudflare.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *WorkersSubdomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkersSubdomainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.Workers.Subdomains.Get(
		ctx,
		workers.SubdomainGetParams{AccountID: cloudflare.F(data.AccountID.ValueString())},
		option.WithMiddleware(logging.Middleware(ctx)),
	)
	if err != nil {
		resp.Diagnostics.AddError("failed to make http request", err.Error())
		return
	}

	data.Subdomain = types.StringValue(result.Subdomain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
