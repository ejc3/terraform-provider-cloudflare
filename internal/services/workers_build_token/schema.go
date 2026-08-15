package workers_build_token

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

var _ resource.ResourceWithConfigValidators = (*WorkersBuildTokenResource)(nil)

// ResourceSchema returns the Terraform schema for a Workers Builds deployment
// token registration.
func ResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Version: 500,
		MarkdownDescription: schemata.Description{
			Scopes: []string{"Workers CI Read", "Workers CI Write"},
		}.String() + "\n\nRegisters a user-owned deployment token with Workers Builds. The provider control-plane token and the registered build/deploy token are separate user-owned credentials. Cloudflare never returns the deployment secret, so it remains in sensitive Terraform state and is never written to provider request logs. Import leaves `build_token_secret` null; the first configured value is recorded locally without remote verification, and later secret changes replace the registration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Build token UUID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"account_id": schema.StringAttribute{
				Description:   "Cloudflare account identifier.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(32, 32),
				},
			},
			"build_token_name": schema.StringAttribute{
				Description:   "Display name for the Workers Builds deployment token.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"build_token_secret": schema.StringAttribute{
				Description: "User-owned Cloudflare API token value used by Workers Builds to deploy. Cloudflare never returns this write-only value.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					requiresReplaceWhenSecretAlreadyInState(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"cloudflare_token_id": schema.StringAttribute{
				Description:   "Identifier of the user-owned Cloudflare API token supplied in build_token_secret.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"owner_type": schema.StringAttribute{
				Description: "Token owner type reported by Cloudflare. Workers Builds currently supports only user-owned tokens.",
				Computed:    true,
			},
		},
	}
}

func (r *WorkersBuildTokenResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ResourceSchema(ctx)
}

func (r *WorkersBuildTokenResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return nil
}

// requiresReplaceWhenSecretAlreadyInState permits exactly one non-remote
// transition: after import, the API cannot return the secret, so state contains
// null and configuration must reconcile the original value. Once a secret is
// present in state, any subsequent change replaces the remote registration.
func requiresReplaceWhenSecretAlreadyInState() planmodifier.String {
	description := "Changing an established build token secret requires replacement; a null imported secret may be reconciled once from configuration."
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull() && !req.StateValue.IsUnknown()
		},
		description,
		description,
	)
}
