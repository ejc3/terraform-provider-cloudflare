package workers_subdomain

import (
	"context"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/schemata"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func ResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Version: 500,
		MarkdownDescription: schemata.Description{
			Scopes: []string{
				"Workers Scripts Read",
				"Workers Scripts Write",
			},
		}.String(),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for the resource. This is the account ID because the Workers subdomain is an account-wide singleton.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_id": schema.StringAttribute{
				Description: "Identifier of the Cloudflare account.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subdomain": schema.StringAttribute{
				Description: "Account-wide label used for workers.dev hostnames. This does not enable workers.dev or preview URLs for any individual Worker.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *WorkersSubdomainResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ResourceSchema(ctx)
}
