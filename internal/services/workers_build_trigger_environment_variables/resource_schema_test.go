package workers_build_trigger_environment_variables_test

import (
	"context"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_build_trigger_environment_variables"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/test_helpers"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkersBuildTriggerEnvironmentVariablesModelSchemaParity(t *testing.T) {
	t.Parallel()
	model := (*workers_build_trigger_environment_variables.WorkersBuildTriggerEnvironmentVariablesModel)(nil)
	resourceSchema := workers_build_trigger_environment_variables.ResourceSchema(context.Background())
	errors := test_helpers.ValidateResourceModelSchemaIntegrity(model, resourceSchema)
	// The hand-written map uses a basetype so secret values can be preserved
	// from prior state explicitly. The semantic test below validates the map
	// and both nested attributes directly.
	errors.Ignore(t, ".@WorkersBuildTriggerEnvironmentVariablesModel.variables")
	errors.Report(t)
}

func TestWorkersBuildTriggerEnvironmentVariablesSchemaSemantics(t *testing.T) {
	t.Parallel()
	resourceSchema := workers_build_trigger_environment_variables.ResourceSchema(context.Background())

	id := resourceSchema.Attributes["id"].(schema.StringAttribute)
	if !id.Computed || id.Required || id.Optional {
		t.Fatalf("id must be computed only: %#v", id)
	}
	for _, name := range []string{"account_id", "trigger_uuid"} {
		attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
		if !attribute.Required || attribute.Optional || attribute.Computed {
			t.Errorf("%s must be required only: %#v", name, attribute)
		}
		assertEnvironmentVariableChangeRequiresReplace(t, resourceSchema, name, attribute)
	}
	assertEnvironmentVariableInvalidString(t, resourceSchema, "account_id", "short")
	assertEnvironmentVariableInvalidString(t, resourceSchema, "account_id", "gggggggggggggggggggggggggggggggg")
	assertEnvironmentVariableInvalidString(t, resourceSchema, "trigger_uuid", "not-a-uuid")
	assertEnvironmentVariableValidString(t, resourceSchema, "trigger_uuid", "33333333-3333-0333-7333-333333333333")

	variables := resourceSchema.Attributes["variables"].(schema.MapNestedAttribute)
	if !variables.Required || variables.Optional || variables.Computed || !variables.Sensitive {
		t.Fatalf("variables must be a required sensitive map: %#v", variables)
	}
	value := variables.NestedObject.Attributes["value"].(schema.StringAttribute)
	if !value.Optional || value.Required || value.Computed || !value.Sensitive {
		t.Fatalf("variables.value must be optional and sensitive for import reconciliation: %#v", value)
	}
	isSecret := variables.NestedObject.Attributes["is_secret"].(schema.BoolAttribute)
	if !isSecret.Required || isSecret.Optional || isSecret.Computed {
		t.Fatalf("variables.is_secret must be required only: %#v", isSecret)
	}
}

func assertEnvironmentVariableValidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
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

func assertEnvironmentVariableInvalidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
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

func environmentVariableSchemaModel(t *testing.T) workers_build_trigger_environment_variables.WorkersBuildTriggerEnvironmentVariablesModel {
	t.Helper()
	ctx := context.Background()
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"value": types.StringType, "is_secret": types.BoolType}}
	variables, diagnostics := types.MapValueFrom(ctx, objectType, map[string]workers_build_trigger_environment_variables.EnvironmentVariableModel{
		"CLOUDFLARE_ACCOUNT_ID": {
			Value:    types.StringValue("12ea67fb7ced068de03f35c22688e436"),
			IsSecret: types.BoolValue(false),
		},
	})
	if diagnostics.HasError() {
		t.Fatalf("make variables map: %v", diagnostics)
	}
	return workers_build_trigger_environment_variables.WorkersBuildTriggerEnvironmentVariablesModel{
		ID:          types.StringValue("33333333-3333-4333-8333-333333333333"),
		AccountID:   types.StringValue("023e105f4ecef8ad9ca31a8372d0c353"),
		TriggerUUID: types.StringValue("33333333-3333-4333-8333-333333333333"),
		Variables:   variables,
	}
}

func assertEnvironmentVariableChangeRequiresReplace(t *testing.T, resourceSchema schema.Schema, name string, attribute schema.StringAttribute) {
	t.Helper()
	ctx := context.Background()
	stateModel := environmentVariableSchemaModel(t)
	planModel := stateModel
	stateValue := types.String{}
	planValue := types.String{}
	switch name {
	case "account_id":
		stateValue = stateModel.AccountID
		planModel.AccountID = types.StringValue("12ea67fb7ced068de03f35c22688e436")
		planValue = planModel.AccountID
	case "trigger_uuid":
		stateValue = stateModel.TriggerUUID
		planModel.TriggerUUID = types.StringValue("44444444-4444-4444-8444-444444444444")
		planModel.ID = planModel.TriggerUUID
		planValue = planModel.TriggerUUID
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
