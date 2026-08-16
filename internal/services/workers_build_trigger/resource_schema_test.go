package workers_build_trigger_test

import (
	"context"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_build_trigger"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/test_helpers"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkersBuildTriggerModelSchemaParity(t *testing.T) {
	t.Parallel()
	model := (*workers_build_trigger.WorkersBuildTriggerModel)(nil)
	resourceSchema := workers_build_trigger.ResourceSchema(context.Background())
	errors := test_helpers.ValidateResourceModelSchemaIntegrity(model, resourceSchema)
	// These hand-written sets deliberately use framework basetypes because the
	// resource owns deterministic set-to-API conversion instead of generated
	// customfield serialization. The semantic test below checks each set deeply.
	for _, name := range []string{"branch_includes", "branch_excludes", "path_includes", "path_excludes"} {
		errors.Ignore(t, ".@WorkersBuildTriggerModel."+name)
	}
	errors.Report(t)
}

func TestWorkersBuildTriggerSchemaSemantics(t *testing.T) {
	t.Parallel()
	resourceSchema := workers_build_trigger.ResourceSchema(context.Background())

	for _, name := range []string{
		"account_id", "external_script_id", "repository_connection_uuid", "build_token_uuid",
		"trigger_name", "build_command", "deploy_command", "root_directory",
		"branch_includes", "branch_excludes", "path_includes", "path_excludes",
		"build_caching_enabled",
	} {
		attribute, exists := resourceSchema.Attributes[name]
		if !exists {
			t.Fatalf("missing required attribute %s", name)
		}
		switch typed := attribute.(type) {
		case schema.StringAttribute:
			if !typed.Required || typed.Optional || typed.Computed {
				t.Errorf("%s must be required only: %#v", name, typed)
			}
		case schema.SetAttribute:
			if !typed.Required || typed.Optional || typed.Computed || !typed.ElementType.Equal(types.StringType) {
				t.Errorf("%s must be a required set of strings: %#v", name, typed)
			}
		case schema.BoolAttribute:
			if !typed.Required || typed.Optional || typed.Computed {
				t.Errorf("%s must be required only: %#v", name, typed)
			}
		default:
			t.Errorf("%s has unexpected schema type %T", name, attribute)
		}
	}

	for _, name := range []string{"id", "created_on", "modified_on"} {
		attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
		if !attribute.Computed || attribute.Required || attribute.Optional {
			t.Errorf("%s must be computed only: %#v", name, attribute)
		}
	}
	if modifiers := resourceSchema.Attributes["modified_on"].(schema.StringAttribute).PlanModifiers; len(modifiers) != 0 {
		t.Fatalf("modified_on must remain unknown during updates, got %d plan modifiers", len(modifiers))
	}

	assertTriggerInvalidString(t, resourceSchema, "account_id", "short")
	assertTriggerInvalidString(t, resourceSchema, "account_id", "gggggggggggggggggggggggggggggggg")
	assertTriggerInvalidString(t, resourceSchema, "external_script_id", "not-a-worker-tag")
	assertTriggerInvalidString(t, resourceSchema, "repository_connection_uuid", "not-a-uuid")
	assertTriggerInvalidString(t, resourceSchema, "build_token_uuid", "not-a-uuid")
	assertTriggerValidString(t, resourceSchema, "repository_connection_uuid", "11111111-1111-0111-7111-111111111111")
	assertTriggerValidString(t, resourceSchema, "build_token_uuid", "22222222-2222-0222-7222-222222222222")

	for _, name := range []string{"account_id", "external_script_id", "repository_connection_uuid"} {
		attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
		assertTriggerChangeRequiresReplace(t, resourceSchema, name, attribute)
	}
}

func assertTriggerValidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
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

func assertTriggerInvalidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
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

func triggerSchemaModel(t *testing.T) workers_build_trigger.WorkersBuildTriggerModel {
	t.Helper()
	ctx := context.Background()
	set, diagnostics := types.SetValueFrom(ctx, types.StringType, []string{"*"})
	if diagnostics.HasError() {
		t.Fatalf("make set: %v", diagnostics)
	}
	empty, diagnostics := types.SetValueFrom(ctx, types.StringType, []string{})
	if diagnostics.HasError() {
		t.Fatalf("make empty set: %v", diagnostics)
	}
	return workers_build_trigger.WorkersBuildTriggerModel{
		ID:                   types.StringValue("33333333-3333-4333-8333-333333333333"),
		AccountID:            types.StringValue("023e105f4ecef8ad9ca31a8372d0c353"),
		ExternalScriptID:     types.StringValue("72edf31f83e240448fce38bef56104e3"),
		RepositoryConnection: types.StringValue("11111111-1111-4111-8111-111111111111"),
		BuildTokenUUID:       types.StringValue("22222222-2222-4222-8222-222222222222"),
		TriggerName:          types.StringValue("preview"),
		BuildCommand:         types.StringValue("npm run cf:build"),
		DeployCommand:        types.StringValue("npm run cf:upload:built"),
		RootDirectory:        types.StringValue("/"),
		BranchIncludes:       set,
		BranchExcludes:       empty,
		PathIncludes:         set,
		PathExcludes:         empty,
		BuildCachingEnabled:  types.BoolValue(true),
		CreatedOn:            types.StringValue("2026-08-15T00:00:00Z"),
		ModifiedOn:           types.StringValue("2026-08-15T00:01:00Z"),
	}
}

func assertTriggerChangeRequiresReplace(t *testing.T, resourceSchema schema.Schema, name string, attribute schema.StringAttribute) {
	t.Helper()
	ctx := context.Background()
	stateModel := triggerSchemaModel(t)
	planModel := stateModel
	stateValue := types.String{}
	planValue := types.String{}
	switch name {
	case "account_id":
		stateValue = stateModel.AccountID
		planModel.AccountID = types.StringValue("12ea67fb7ced068de03f35c22688e436")
		planValue = planModel.AccountID
	case "external_script_id":
		stateValue = stateModel.ExternalScriptID
		planModel.ExternalScriptID = types.StringValue("82edf31f83e240448fce38bef56104e3")
		planValue = planModel.ExternalScriptID
	case "repository_connection_uuid":
		stateValue = stateModel.RepositoryConnection
		planModel.RepositoryConnection = types.StringValue("44444444-4444-4444-8444-444444444444")
		planValue = planModel.RepositoryConnection
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
