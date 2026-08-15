package workers_build_repository_connection_test

import (
	"context"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_build_repository_connection"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/test_helpers"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkersBuildRepositoryConnectionModelSchemaParity(t *testing.T) {
	t.Parallel()
	model := (*workers_build_repository_connection.WorkersBuildRepositoryConnectionModel)(nil)
	resourceSchema := workers_build_repository_connection.ResourceSchema(context.Background())
	errors := test_helpers.ValidateResourceModelSchemaIntegrity(model, resourceSchema)
	errors.Report(t)
}

func TestWorkersBuildRepositoryConnectionSchemaSemantics(t *testing.T) {
	t.Parallel()
	resourceSchema := workers_build_repository_connection.ResourceSchema(context.Background())

	for _, name := range []string{"account_id", "provider_type", "provider_account_id", "provider_account_name", "repo_id", "repo_name"} {
		attribute, ok := resourceSchema.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s is not a string attribute", name)
		}
		if !attribute.Required {
			t.Errorf("%s must be required", name)
		}
		if len(attribute.PlanModifiers) == 0 {
			t.Errorf("%s must require replacement", name)
		}
		assertRepositoryConnectionChangeRequiresReplace(t, resourceSchema, name, attribute)
	}

	providerType := resourceSchema.Attributes["provider_type"].(schema.StringAttribute)
	var validationResponse validator.StringResponse
	for _, valueValidator := range providerType.Validators {
		valueValidator.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("provider_type"),
			ConfigValue: types.StringValue("bitbucket"),
		}, &validationResponse)
	}
	if !validationResponse.Diagnostics.HasError() {
		t.Fatal("provider_type must reject unsupported providers")
	}
	assertRepositoryConnectionValidString(t, resourceSchema, "provider_type", "github")
	assertRepositoryConnectionValidString(t, resourceSchema, "account_id", "023e105f4ecef8ad9ca31a8372d0c353")
	assertRepositoryConnectionInvalidString(t, resourceSchema, "account_id", "gggggggggggggggggggggggggggggggg")

	for _, name := range []string{"provider_account_id", "repo_id"} {
		attribute := resourceSchema.Attributes[name].(schema.StringAttribute)
		var response validator.StringResponse
		for _, valueValidator := range attribute.Validators {
			valueValidator.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root(name),
				ConfigValue: types.StringValue("not-numeric"),
			}, &response)
		}
		if !response.Diagnostics.HasError() {
			t.Fatalf("%s must reject non-numeric identifiers", name)
		}
	}
	assertRepositoryConnectionValidString(t, resourceSchema, "provider_account_id", "250920182")
	assertRepositoryConnectionValidString(t, resourceSchema, "repo_id", "1120877379")
}

func assertRepositoryConnectionValidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
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

func assertRepositoryConnectionInvalidString(t *testing.T, resourceSchema schema.Schema, name, value string) {
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

func assertRepositoryConnectionChangeRequiresReplace(t *testing.T, resourceSchema schema.Schema, name string, attribute schema.StringAttribute) {
	t.Helper()
	ctx := context.Background()
	stateModel := workers_build_repository_connection.WorkersBuildRepositoryConnectionModel{
		ID:                  types.StringValue("11111111-1111-4111-8111-111111111111"),
		AccountID:           types.StringValue("023e105f4ecef8ad9ca31a8372d0c353"),
		ProviderType:        types.StringValue("github"),
		ProviderAccountID:   types.StringValue("250920182"),
		ProviderAccountName: types.StringValue("CoderColton"),
		RepoID:              types.StringValue("1120877379"),
		RepoName:            types.StringValue("colton-games"),
	}
	planModel := stateModel
	stateValue := types.String{}
	planValue := types.String{}
	switch name {
	case "account_id":
		stateValue = stateModel.AccountID
		planModel.AccountID = types.StringValue("12ea67fb7ced068de03f35c22688e436")
		planValue = planModel.AccountID
	case "provider_type":
		stateValue = stateModel.ProviderType
		planModel.ProviderType = types.StringValue("gitlab")
		planValue = planModel.ProviderType
	case "provider_account_id":
		stateValue = stateModel.ProviderAccountID
		planModel.ProviderAccountID = types.StringValue("250920183")
		planValue = planModel.ProviderAccountID
	case "provider_account_name":
		stateValue = stateModel.ProviderAccountName
		planModel.ProviderAccountName = types.StringValue("DifferentOwner")
		planValue = planModel.ProviderAccountName
	case "repo_id":
		stateValue = stateModel.RepoID
		planModel.RepoID = types.StringValue("1120877380")
		planValue = planModel.RepoID
	case "repo_name":
		stateValue = stateModel.RepoName
		planModel.RepoName = types.StringValue("different-repo")
		planValue = planModel.RepoName
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
