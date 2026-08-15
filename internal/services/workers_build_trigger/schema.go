package workers_build_trigger

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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var workerTagPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

func ResourceSchema(_ context.Context) schema.Schema {
	uuidValidators := []validator.String{stringvalidator.RegexMatches(uuidPattern, "must be a UUID")}
	return schema.Schema{
		Version: 500,
		MarkdownDescription: schemata.Description{
			Scopes: []string{"Workers CI Write"},
		}.String() + "\n\nManages a typed Cloudflare Workers Builds production or preview trigger.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Trigger UUID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account_id": schema.StringAttribute{
				Description:   "Cloudflare account identifier.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.RegexMatches(workerTagPattern, "must be a 32-character hexadecimal account identifier")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"external_script_id": schema.StringAttribute{
				Description:   "Immutable Worker tag, not the Worker name.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.RegexMatches(workerTagPattern, "must be the 32-character hexadecimal Worker tag")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository_connection_uuid": schema.StringAttribute{
				Description:   "Workers Builds repository connection UUID.",
				Required:      true,
				Validators:    uuidValidators,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"build_token_uuid": schema.StringAttribute{
				Description: "Workers Builds deployment token UUID.",
				Required:    true,
				Validators:  uuidValidators,
			},
			"trigger_name": schema.StringAttribute{
				Description: "Display name for the trigger.", Required: true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"build_command": schema.StringAttribute{
				Description: "Command used to build the Worker.", Required: true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"deploy_command": schema.StringAttribute{
				Description: "Command used to deploy or upload the built Worker.", Required: true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"root_directory": schema.StringAttribute{
				Description: "Repository directory in which commands run.", Required: true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"branch_includes": stringSetAttribute("Branch patterns which trigger a build."),
			"branch_excludes": stringSetAttribute("Branch patterns excluded from builds."),
			"path_includes":   stringSetAttribute("Path patterns which trigger a build."),
			"path_excludes":   stringSetAttribute("Path patterns excluded from builds."),
			"build_caching_enabled": schema.BoolAttribute{
				Description: "Whether Workers Builds may reuse its build cache.",
				Required:    true,
			},
			"created_on": schema.StringAttribute{
				Description:   "Creation timestamp reported by Cloudflare.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified_on": schema.StringAttribute{
				Description: "Last modification timestamp reported by Cloudflare.",
				Computed:    true,
			},
		},
	}
}

func stringSetAttribute(description string) schema.SetAttribute {
	return schema.SetAttribute{
		Description: description,
		Required:    true,
		ElementType: types.StringType,
	}
}

func (r *WorkersBuildTriggerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ResourceSchema(ctx)
}
