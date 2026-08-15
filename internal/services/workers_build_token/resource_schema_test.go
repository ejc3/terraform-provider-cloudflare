package workers_build_token_test

import (
	"context"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_build_token"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/test_helpers"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkersBuildTokenModelSchemaParity(t *testing.T) {
	t.Parallel()
	model := (*workers_build_token.WorkersBuildTokenModel)(nil)
	resourceSchema := workers_build_token.ResourceSchema(context.Background())
	errors := test_helpers.ValidateResourceModelSchemaIntegrity(model, resourceSchema)
	errors.Report(t)
}

func TestWorkersBuildTokenSchemaSecretSemantics(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resourceSchema := workers_build_token.ResourceSchema(ctx)
	secret := resourceSchema.Attributes["build_token_secret"].(schema.StringAttribute)
	if !secret.Required || !secret.Sensitive {
		t.Fatalf("build_token_secret must be required and sensitive: %#v", secret)
	}
	if len(secret.PlanModifiers) != 1 {
		t.Fatalf("build_token_secret plan modifiers = %d, want 1", len(secret.PlanModifiers))
	}

	stateModel := tokenModel(tokenTestUUID, types.StringNull())
	state := tfsdk.State{Schema: resourceSchema}
	if diagnostics := state.Set(ctx, &stateModel); diagnostics.HasError() {
		t.Fatalf("set state: %v", diagnostics)
	}
	planModel := tokenModel(tokenTestUUID, types.StringValue("configured-secret"))
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diagnostics := plan.Set(ctx, &planModel); diagnostics.HasError() {
		t.Fatalf("set plan: %v", diagnostics)
	}

	response := &planmodifier.StringResponse{}
	secret.PlanModifiers[0].PlanModifyString(ctx, planmodifier.StringRequest{
		State:       state,
		Plan:        plan,
		StateValue:  types.StringNull(),
		PlanValue:   types.StringValue("configured-secret"),
		ConfigValue: types.StringValue("configured-secret"),
	}, response)
	if response.RequiresReplace {
		t.Fatal("the first secret reconciliation after import must not replace the remote token")
	}

	establishedModel := tokenModel(tokenTestUUID, types.StringValue("old-secret"))
	established := tfsdk.State{Schema: resourceSchema}
	if diagnostics := established.Set(ctx, &establishedModel); diagnostics.HasError() {
		t.Fatalf("set established state: %v", diagnostics)
	}
	newPlanModel := tokenModel(tokenTestUUID, types.StringValue("new-secret"))
	newPlan := tfsdk.Plan{Schema: resourceSchema}
	if diagnostics := newPlan.Set(ctx, &newPlanModel); diagnostics.HasError() {
		t.Fatalf("set new plan: %v", diagnostics)
	}
	response = &planmodifier.StringResponse{}
	secret.PlanModifiers[0].PlanModifyString(ctx, planmodifier.StringRequest{
		State:       established,
		Plan:        newPlan,
		StateValue:  types.StringValue("old-secret"),
		PlanValue:   types.StringValue("new-secret"),
		ConfigValue: types.StringValue("new-secret"),
	}, response)
	if !response.RequiresReplace {
		t.Fatal("changing an established secret must replace the remote token")
	}
}

func TestWorkersBuildTokenSchemaIdentitySemantics(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resourceSchema := workers_build_token.ResourceSchema(ctx)

	for _, name := range []string{"id", "owner_type"} {
		attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
		if !attribute.Computed || attribute.Required || attribute.Optional {
			t.Errorf("%s must be computed only: %#v", name, attribute)
		}
	}
	for _, name := range []string{"account_id", "build_token_name", "cloudflare_token_id"} {
		attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
		if !attribute.Required || attribute.Optional || attribute.Computed {
			t.Errorf("%s must be required only: %#v", name, attribute)
		}
		assertBuildTokenChangeRequiresReplace(t, resourceSchema, name, attribute)
	}
	assertBuildTokenInvalidString(t, resourceSchema, "account_id", "short")
	assertBuildTokenInvalidString(t, resourceSchema, "account_id", "gggggggggggggggggggggggggggggggg")
	assertBuildTokenValidString(t, resourceSchema, "account_id", tokenTestAccountID)
	assertBuildTokenInvalidString(t, resourceSchema, "build_token_name", "")
	assertBuildTokenInvalidString(t, resourceSchema, "cloudflare_token_id", "")
}

func assertBuildTokenValidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
	t.Helper()
	attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
	response := &validator.StringResponse{}
	for _, configuredValidator := range attribute.Validators {
		configuredValidator.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root(name),
			ConfigValue: types.StringValue(value),
		}, response)
	}
	if response.Diagnostics.HasError() {
		t.Fatalf("%s unexpectedly rejected %q: %v", name, value, response.Diagnostics)
	}
}

func assertBuildTokenInvalidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
	t.Helper()
	attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
	response := &validator.StringResponse{}
	for _, configuredValidator := range attribute.Validators {
		configuredValidator.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root(name),
			ConfigValue: types.StringValue(value),
		}, response)
	}
	if !response.Diagnostics.HasError() {
		t.Fatalf("%s unexpectedly accepted %q", name, value)
	}
}

func assertBuildTokenChangeRequiresReplace(t *testing.T, resourceSchema schema.Schema, name string, attribute schema.StringAttribute) {
	t.Helper()
	ctx := context.Background()
	stateModel := tokenModel(tokenTestUUID, types.StringValue("established-secret"))
	planModel := stateModel
	stateValue := types.String{}
	planValue := types.String{}
	switch name {
	case "account_id":
		stateValue = stateModel.AccountID
		planModel.AccountID = types.StringValue("023e105f4ecef8ad9ca31a8372d0c353")
		planValue = planModel.AccountID
	case "build_token_name":
		stateValue = stateModel.BuildTokenName
		planModel.BuildTokenName = types.StringValue("different-token")
		planValue = planModel.BuildTokenName
	case "cloudflare_token_id":
		stateValue = stateModel.CloudflareTokenID
		planModel.CloudflareTokenID = types.StringValue("different-token-id")
		planValue = planModel.CloudflareTokenID
	default:
		t.Fatalf("unhandled attribute %q", name)
	}

	state := tfsdk.State{Schema: resourceSchema}
	if diagnostics := state.Set(ctx, &stateModel); diagnostics.HasError() {
		t.Fatalf("set state: %v", diagnostics)
	}
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diagnostics := plan.Set(ctx, &planModel); diagnostics.HasError() {
		t.Fatalf("set plan: %v", diagnostics)
	}
	request := planmodifier.StringRequest{
		State:       state,
		Plan:        plan,
		StateValue:  stateValue,
		PlanValue:   planValue,
		ConfigValue: planValue,
	}
	response := &planmodifier.StringResponse{}
	for _, modifier := range attribute.PlanModifiers {
		modifier.PlanModifyString(ctx, request, response)
	}
	if !response.RequiresReplace {
		t.Fatalf("changing %s must require replacement", name)
	}
}
