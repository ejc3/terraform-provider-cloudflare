package workers_build_trigger_environment_variables

import (
	"context"
	"regexp"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/schemata"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var accountIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

func ResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Version: 500,
		MarkdownDescription: schemata.Description{
			Scopes: []string{"Workers CI Read", "Workers CI Write"},
		}.String() + "\n\nManages the complete set of build-time environment variables for a Workers Builds trigger. Secret values remain in sensitive Terraform state because Cloudflare redacts them on reads.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Trigger UUID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account_id": schema.StringAttribute{
				Description:   "Cloudflare account identifier.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.RegexMatches(accountIDPattern, "must be a 32-character hexadecimal account identifier")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"trigger_uuid": schema.StringAttribute{
				Description:   "Workers Builds trigger UUID.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.RegexMatches(uuidPattern, "must be a UUID")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"variables": schema.MapNestedAttribute{
				Description: "Environment-variable names mapped to sensitive values and their Cloudflare secret flag. This resource owns the complete map.",
				Required:    true,
				Sensitive:   true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"value": schema.StringAttribute{
						Description: "Variable value. It may be null immediately after import for a redacted secret and must then be provided in configuration.",
						Required:    true,
						Sensitive:   true,
					},
					"is_secret": schema.BoolAttribute{
						Description: "Whether Cloudflare masks this value in build logs and subsequent API reads.",
						Required:    true,
					},
				}},
			},
		},
	}
}

func (r *WorkersBuildTriggerEnvironmentVariablesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ResourceSchema(ctx)
}
