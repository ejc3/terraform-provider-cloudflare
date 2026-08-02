// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package zero_trust_access_ai_controls_mcp_portal

import (
	"context"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/customfield"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/schemata"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ resource.ResourceWithConfigValidators = (*ZeroTrustAccessAIControlsMcpPortalResource)(nil)

func ResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: schemata.Description{
			Scopes: []string{
				"MCP Portals Read",
				"MCP Portals Write",
			},
		}.String(),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "portal id",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"account_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"hostname": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"allow_code_mode": schema.BoolAttribute{
				Description:        "Deprecated: use `code_mode` instead. Legacy on/off toggle for Dynamic Workers (codemode). `true` maps to any non-off `code_mode`; `false` maps to `code_mode: off`.",
				Optional:           true,
				DeprecationMessage: "This attribute is deprecated.",
			},
			"code_mode": schema.StringAttribute{
				Description: "Controls Dynamic Workers (codemode) availability for this portal. `off` disables codemode. `opt_in` makes it available but clients must opt in per session. `default_on` enables it by default with a client override. `enforced` requires codemode for every session with no override.\nAvailable values: \"off\", \"opt_in\", \"default_on\", \"enforced\".",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"off",
						"opt_in",
						"default_on",
						"enforced",
					),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"secure_web_gateway": schema.BoolAttribute{
				Description: "Route outbound MCP traffic through Zero Trust Secure Web Gateway",
				Computed:    true,
				Optional:    true,
				Default:     booldefault.StaticBool(false),
			},
			"servers": schema.SetNestedAttribute{
				Computed:   true,
				Optional:   true,
				CustomType: customfield.NewNestedObjectSetType[ZeroTrustAccessAIControlsMcpPortalServersModel](ctx),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"server_id": schema.StringAttribute{
							Description: "server id",
							Required:    true,
						},
						"default_disabled": schema.BoolAttribute{
							Computed: true,
							Optional: true,
							Default:  booldefault.StaticBool(false),
						},
						"on_behalf": schema.BoolAttribute{
							Computed: true,
							Optional: true,
							Default:  booldefault.StaticBool(true),
						},
						"updated_prompts": schema.ListNestedAttribute{
							Optional: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Required: true,
									},
									"alias": schema.StringAttribute{
										Optional: true,
									},
									"description": schema.StringAttribute{
										Optional: true,
									},
									"enabled": schema.BoolAttribute{
										Optional: true,
									},
								},
							},
						},
						"updated_tools": schema.ListNestedAttribute{
							Optional: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Required: true,
									},
									"alias": schema.StringAttribute{
										Optional: true,
									},
									"description": schema.StringAttribute{
										Optional: true,
									},
									"enabled": schema.BoolAttribute{
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Computed:   true,
				CustomType: timetypes.RFC3339Type{},
			},
			"created_by": schema.StringAttribute{
				Computed: true,
			},
			"modified_at": schema.StringAttribute{
				Computed:   true,
				CustomType: timetypes.RFC3339Type{},
			},
			"modified_by": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *ZeroTrustAccessAIControlsMcpPortalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ResourceSchema(ctx)
}

func (r *ZeroTrustAccessAIControlsMcpPortalResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{}
}
