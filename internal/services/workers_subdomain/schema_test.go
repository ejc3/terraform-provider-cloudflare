package workers_subdomain_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_script_subdomain"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_subdomain"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/test_helpers"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkersSubdomainResourceModelSchemaParity(t *testing.T) {
	t.Parallel()
	model := (*workers_subdomain.WorkersSubdomainModel)(nil)
	schema := workers_subdomain.ResourceSchema(context.Background())
	errs := test_helpers.ValidateResourceModelSchemaIntegrity(model, schema)
	errs.Report(t)
}

func TestWorkersSubdomainDataSourceModelSchemaParity(t *testing.T) {
	t.Parallel()
	model := (*workers_subdomain.WorkersSubdomainDataSourceModel)(nil)
	schema := workers_subdomain.DataSourceSchema(context.Background())
	errs := test_helpers.ValidateDataSourceModelSchemaIntegrity(model, schema)
	errs.Report(t)
}

func TestWorkersSubdomainSchemaSemantics(t *testing.T) {
	t.Parallel()
	schema := workers_subdomain.ResourceSchema(context.Background())

	accountID, ok := schema.Attributes["account_id"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("account_id is not a string attribute")
	}
	if !accountID.Required || accountID.Optional || accountID.Computed {
		t.Fatalf("account_id must be required only: %#v", accountID)
	}
	if len(accountID.PlanModifiers) == 0 {
		t.Fatal("account_id must require replacement when changed")
	}
	prior := resourceState(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("cc-games"),
	})
	nextAccountID := strings.Repeat("a", 32)
	plan := resourcePlan(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(nextAccountID),
		Subdomain: types.StringValue("cc-games"),
	})
	modifierRequest := planmodifier.StringRequest{
		State:      prior,
		Plan:       plan,
		StateValue: types.StringValue(testAccountID),
		PlanValue:  types.StringValue(nextAccountID),
	}
	modifierResponse := &planmodifier.StringResponse{}
	for _, modifier := range accountID.PlanModifiers {
		modifier.PlanModifyString(context.Background(), modifierRequest, modifierResponse)
	}
	if !modifierResponse.RequiresReplace {
		t.Fatal("changing account_id must require replacement")
	}
	assertInvalidString(t, accountID.Validators, "")
	assertInvalidString(t, accountID.Validators, strings.Repeat("a", 33))
	assertInvalidString(t, accountID.Validators, strings.Repeat("g", 32))
	assertValidString(t, accountID.Validators, testAccountID)

	subdomain, ok := schema.Attributes["subdomain"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("subdomain is not a string attribute")
	}
	if !subdomain.Required || subdomain.Optional || subdomain.Computed {
		t.Fatalf("subdomain must be required only: %#v", subdomain)
	}
	assertInvalidString(t, subdomain.Validators, "")

	id, ok := schema.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !id.Computed {
		t.Fatal("id must be a computed string attribute")
	}
}

func TestWorkersSubdomainDataSourceSchemaSemantics(t *testing.T) {
	t.Parallel()
	schema := workers_subdomain.DataSourceSchema(context.Background())
	accountID := schema.Attributes["account_id"].(datasourceschema.StringAttribute)
	if !accountID.Required || accountID.Optional || accountID.Computed {
		t.Fatalf("data source account_id must be required only: %#v", accountID)
	}
	assertInvalidString(t, accountID.Validators, "")
	assertInvalidString(t, accountID.Validators, strings.Repeat("a", 33))
	assertInvalidString(t, accountID.Validators, strings.Repeat("g", 32))
	assertValidString(t, accountID.Validators, testAccountID)
	subdomain := schema.Attributes["subdomain"].(datasourceschema.StringAttribute)
	if !subdomain.Computed || subdomain.Optional || subdomain.Required {
		t.Fatalf("data source subdomain must be computed only: %#v", subdomain)
	}
}

func assertInvalidString(t *testing.T, validators []validator.String, value string) {
	t.Helper()
	if len(validators) == 0 {
		t.Fatal("expected at least one string validator")
	}
	for _, configuredValidator := range validators {
		resp := &validator.StringResponse{}
		configuredValidator.ValidateString(
			context.Background(),
			validator.StringRequest{ConfigValue: types.StringValue(value)},
			resp,
		)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	t.Fatalf("value %q unexpectedly passed validation", value)
}

func assertValidString(t *testing.T, validators []validator.String, value string) {
	t.Helper()
	if len(validators) == 0 {
		t.Fatal("expected at least one string validator")
	}
	for _, configuredValidator := range validators {
		resp := &validator.StringResponse{}
		configuredValidator.ValidateString(
			context.Background(),
			validator.StringRequest{ConfigValue: types.StringValue(value)},
			resp,
		)
		if resp.Diagnostics.HasError() {
			t.Fatalf("value %q unexpectedly failed validation: %v", value, resp.Diagnostics)
		}
	}
}

func TestAccountLabelAndPerScriptSwitchesRemainIndependent(t *testing.T) {
	t.Parallel()
	accountSchema := workers_subdomain.ResourceSchema(context.Background())
	scriptSchema := workers_script_subdomain.ResourceSchema(context.Background())

	for _, name := range []string{"enabled", "previews_enabled", "script_name"} {
		if _, exists := accountSchema.Attributes[name]; exists {
			t.Errorf("account-wide Workers subdomain unexpectedly owns per-script attribute %q", name)
		}
	}
	if _, exists := accountSchema.Attributes["subdomain"]; !exists {
		t.Error("account-wide Workers subdomain must own the account label")
	}
	if _, exists := scriptSchema.Attributes["subdomain"]; exists {
		t.Error("per-script Workers subdomain must not own the account label")
	}
	for _, name := range []string{"enabled", "previews_enabled", "script_name"} {
		if _, exists := scriptSchema.Attributes[name]; !exists {
			t.Errorf("per-script Workers subdomain is missing %q", name)
		}
	}
}
