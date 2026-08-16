package workers_subdomain

import (
	"context"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/schemata"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func DataSourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: schemata.Description{
			Scopes: []string{"Workers Scripts Read"},
		}.String(),
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Description: "Identifier of the Cloudflare account.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(accountIDPattern, "must be a 32-character hexadecimal account identifier"),
				},
			},
			"subdomain": schema.StringAttribute{
				Description: "Account-wide label used for workers.dev hostnames.",
				Computed:    true,
			},
		},
	}
}

func (d *WorkersSubdomainDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = DataSourceSchema(ctx)
}
