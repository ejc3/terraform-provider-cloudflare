package workers_build_repository_connection

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

var numericIDPattern = regexp.MustCompile(`^[0-9]+$`)
var accountIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

var _ resource.ResourceWithConfigValidators = (*WorkersBuildRepositoryConnectionResource)(nil)

// ResourceSchema returns the Terraform schema for a Workers Builds repository
// connection.
func ResourceSchema(_ context.Context) schema.Schema {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	return schema.Schema{
		Version: 500,
		MarkdownDescription: schemata.Description{
			Scopes: []string{"Workers CI Write"},
		}.String() + "\n\nManages a typed Workers Builds repository connection. Cloudflare exposes no read or list endpoint for this object, so refresh preserves the last confirmed state and import is intentionally unsupported. For private GitHub repositories, install and authorize the Cloudflare Workers and Pages GitHub App before creating the connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Repository connection UUID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"account_id": schema.StringAttribute{
				Description:   "Cloudflare account identifier.",
				Required:      true,
				PlanModifiers: requiresReplace,
				Validators: []validator.String{
					stringvalidator.RegexMatches(accountIDPattern, "must be a 32-character hexadecimal account identifier"),
				},
			},
			"provider_type": schema.StringAttribute{
				Description:   "Git provider type.",
				Required:      true,
				PlanModifiers: requiresReplace,
				Validators: []validator.String{
					stringvalidator.OneOf("github", "gitlab", "gitlab_internal"),
				},
			},
			"provider_account_id": schema.StringAttribute{
				Description:   "Provider account numeric identifier, represented as a string to preserve precision.",
				Required:      true,
				PlanModifiers: requiresReplace,
				Validators: []validator.String{
					stringvalidator.RegexMatches(numericIDPattern, "must contain only decimal digits"),
				},
			},
			"provider_account_name": schema.StringAttribute{
				Description:   "Provider account name.",
				Required:      true,
				PlanModifiers: requiresReplace,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"repo_id": schema.StringAttribute{
				Description:   "Repository numeric identifier, represented as a string to preserve precision.",
				Required:      true,
				PlanModifiers: requiresReplace,
				Validators: []validator.String{
					stringvalidator.RegexMatches(numericIDPattern, "must contain only decimal digits"),
				},
			},
			"repo_name": schema.StringAttribute{
				Description:   "Repository name.",
				Required:      true,
				PlanModifiers: requiresReplace,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *WorkersBuildRepositoryConnectionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ResourceSchema(ctx)
}

func (r *WorkersBuildRepositoryConnectionResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return nil
}
