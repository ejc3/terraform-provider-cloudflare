package workers_build_trigger_environment_variables

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	envHTTPTestAccountID   = "023e105f4ecef8ad9ca31a8372d0c353"
	envHTTPTestTriggerUUID = "33333333-3333-4333-8333-333333333333"
	envHTTPTestSecret      = "super-secret-value-must-not-leak"
)

func TestMergeAPIStatePreservesRedactedSecret(t *testing.T) {
	ctx := context.Background()
	secret := EnvironmentVariableModel{Value: types.StringValue("do-not-log-me"), IsSecret: types.BoolValue(true)}
	data := WorkersBuildTriggerEnvironmentVariablesModel{}
	diags := mergeAPIState(ctx, &data, map[string]EnvironmentVariableModel{"API_KEY": secret}, map[string]environmentVariableResponse{
		"API_KEY": {Value: nil, IsSecret: true},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var values map[string]EnvironmentVariableModel
	if diags := data.Variables.ElementsAs(ctx, &values, false); diags.HasError() {
		t.Fatalf("failed to decode state: %v", diags)
	}
	if got := values["API_KEY"].Value.ValueString(); got != "do-not-log-me" {
		t.Fatalf("redacted secret was not preserved, got %q", got)
	}
}

func TestMergeAPIStateLeavesImportedSecretNull(t *testing.T) {
	ctx := context.Background()
	data := WorkersBuildTriggerEnvironmentVariablesModel{}
	diags := mergeAPIState(ctx, &data, nil, map[string]environmentVariableResponse{
		"API_KEY": {Value: nil, IsSecret: true},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	values := data.Variables.Elements()
	object, ok := values["API_KEY"].(types.Object)
	if !ok {
		t.Fatalf("expected object value, got %T", values["API_KEY"])
	}
	if !object.Attributes()["value"].(types.String).IsNull() {
		t.Fatal("an imported redacted secret must remain null until configured")
	}
	if !object.Type(ctx).Equal(types.ObjectType{AttrTypes: map[string]attr.Type{"value": types.StringType, "is_secret": types.BoolType}}) {
		t.Fatal("unexpected object type")
	}
}

func TestMergeDesiredAPIStateIgnoresUndeclaredRemoteVariables(t *testing.T) {
	ctx := context.Background()
	data := WorkersBuildTriggerEnvironmentVariablesModel{}
	desired := map[string]EnvironmentVariableModel{
		"NODE_ENV": {Value: types.StringValue("old"), IsSecret: types.BoolValue(false)},
	}
	diags := mergeDesiredAPIState(ctx, &data, desired, map[string]environmentVariableResponse{
		"NODE_ENV": {Value: stringPointer("production"), IsSecret: false},
		"EXTRA":    {Value: stringPointer("remote-only"), IsSecret: false},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	values := envHTTPTestDecodeVariables(t, data.Variables)
	if len(values) != 1 || values["NODE_ENV"].Value.ValueString() != "production" {
		t.Fatalf("undeclared remote variable entered state: %#v", values)
	}
}

func TestMergeDesiredAPIStatePreservesRedactedSecretValue(t *testing.T) {
	ctx := context.Background()
	data := WorkersBuildTriggerEnvironmentVariablesModel{}
	desired := map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	}
	api := map[string]environmentVariableResponse{
		"API_KEY": {Value: nil, IsSecret: true},
	}
	if diags := mergeDesiredAPIState(ctx, &data, desired, api); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	values := envHTTPTestDecodeVariables(t, data.Variables)
	if len(values) != 1 || values["API_KEY"].Value.ValueString() != envHTTPTestSecret {
		t.Fatalf("authoritative GET erased desired secret state: %#v", values)
	}
}

func TestMergeDesiredAPIStateRejectsUnverifiedDesiredVariable(t *testing.T) {
	ctx := context.Background()
	tests := map[string]struct {
		desired EnvironmentVariableModel
		api     map[string]environmentVariableResponse
	}{
		"missing variable": {
			desired: EnvironmentVariableModel{Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
			api:     map[string]environmentVariableResponse{},
		},
		"secret kind mismatch": {
			desired: EnvironmentVariableModel{Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
			api: map[string]environmentVariableResponse{
				"API_KEY": {Value: stringPointer(envHTTPTestSecret), IsSecret: false},
			},
		},
		"missing non-secret value": {
			desired: EnvironmentVariableModel{Value: types.StringValue("production"), IsSecret: types.BoolValue(false)},
			api: map[string]environmentVariableResponse{
				"API_KEY": {Value: nil, IsSecret: false},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			data := WorkersBuildTriggerEnvironmentVariablesModel{}
			desired := map[string]EnvironmentVariableModel{"API_KEY": test.desired}
			diags := mergeDesiredAPIState(ctx, &data, desired, test.api)
			if !diags.HasError() {
				t.Fatal("unverified desired variable must produce an error diagnostic")
			}
			if strings.Contains(fmt.Sprint(diags), envHTTPTestSecret) {
				t.Fatalf("verification diagnostic leaked secret: %v", diags)
			}
			if !data.Variables.IsNull() {
				t.Fatalf("unverified desired variable synthesized state: %s", data.Variables)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func TestEnvironmentVariablesResponseEnvelope(t *testing.T) {
	payload := []byte(`{"success":true,"result":{"API_KEY":{"created_on":"2026-08-15T00:00:00Z","is_secret":true,"value":null},"NODE_ENV":{"created_on":"2026-08-15T00:00:00Z","is_secret":false,"value":"production"}}}`)
	var envelope environmentVariablesEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || !envelope.Result["API_KEY"].IsSecret || envelope.Result["API_KEY"].Value != nil {
		t.Fatalf("secret response was not decoded safely: %#v", envelope.Result["API_KEY"])
	}
	if got := *envelope.Result["NODE_ENV"].Value; got != "production" {
		t.Fatalf("unexpected plain value %q", got)
	}
}

func TestEnvironmentVariablesEnvelopeRejectsErrorsWithoutEchoingMessage(t *testing.T) {
	err := validateEnvelope(true, []apiError{{Code: 9109, Message: envHTTPTestSecret}})
	if err == nil {
		t.Fatal("an envelope containing errors must be rejected")
	}
	if strings.Contains(err.Error(), envHTTPTestSecret) || !strings.Contains(err.Error(), "9109") {
		t.Fatalf("unsafe or incomplete envelope error: %v", err)
	}
}

func envHTTPTestMap(t *testing.T, values map[string]EnvironmentVariableModel) types.Map {
	t.Helper()
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"value": types.StringType, "is_secret": types.BoolType}}
	result, diags := types.MapValueFrom(context.Background(), objectType, values)
	if diags.HasError() {
		t.Fatalf("make environment-variable map: %v", diags)
	}
	return result
}

func envHTTPTestModel(t *testing.T, values map[string]EnvironmentVariableModel) WorkersBuildTriggerEnvironmentVariablesModel {
	t.Helper()
	return WorkersBuildTriggerEnvironmentVariablesModel{
		ID:          types.StringValue(envHTTPTestTriggerUUID),
		AccountID:   types.StringValue(envHTTPTestAccountID),
		TriggerUUID: types.StringValue(envHTTPTestTriggerUUID),
		Variables:   envHTTPTestMap(t, values),
	}
}

func envHTTPTestState(t *testing.T, model WorkersBuildTriggerEnvironmentVariablesModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: ResourceSchema(context.Background())}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set environment-variable state: %v", diags)
	}
	return state
}

func envHTTPTestPlan(t *testing.T, model WorkersBuildTriggerEnvironmentVariablesModel) tfsdk.Plan {
	t.Helper()
	state := envHTTPTestState(t, model)
	return tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
}

func envHTTPTestInvalidVariablesPlan(t *testing.T) tfsdk.Plan {
	t.Helper()
	invalidSchema := resourceschema.Schema{Attributes: map[string]resourceschema.Attribute{
		"id":           resourceschema.StringAttribute{Computed: true},
		"account_id":   resourceschema.StringAttribute{Required: true},
		"trigger_uuid": resourceschema.StringAttribute{Required: true},
		"variables": resourceschema.MapAttribute{
			Required:    true,
			ElementType: types.StringType,
		},
	}}
	model := WorkersBuildTriggerEnvironmentVariablesModel{
		ID:          types.StringValue(envHTTPTestTriggerUUID),
		AccountID:   types.StringValue(envHTTPTestAccountID),
		TriggerUUID: types.StringValue(envHTTPTestTriggerUUID),
		Variables: types.MapValueMust(types.StringType, map[string]attr.Value{
			"API_KEY": types.StringValue("not-an-environment-variable-object"),
		}),
	}
	state := tfsdk.State{Schema: invalidSchema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set invalid environment-variable plan: %v", diags)
	}
	return tfsdk.Plan{Schema: invalidSchema, Raw: state.Raw}
}

func envHTTPTestResource(t *testing.T, baseURL string) *WorkersBuildTriggerEnvironmentVariablesResource {
	t.Helper()
	r := NewResource().(*WorkersBuildTriggerEnvironmentVariablesResource)
	client := cloudflare.NewClient(option.WithBaseURL(baseURL), option.WithAPIToken("test-control-plane-token"))
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("configure environment-variable resource: %v", resp.Diagnostics)
	}
	return r
}

func envHTTPTestPath() string {
	return "/accounts/" + envHTTPTestAccountID + "/builds/triggers/" + envHTTPTestTriggerUUID + "/environment_variables"
}

func envHTTPTestWriteJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func envHTTPTestEnvelope() string {
	return `{"success":true,"errors":[],"result":{` +
		`"API_KEY":{"created_on":"2026-08-15T00:00:00Z","is_secret":true,"value":null},` +
		`"NODE_ENV":{"created_on":"2026-08-15T00:00:00Z","is_secret":false,"value":"production"}` +
		`}}`
}

func envHTTPTestDecodeState(t *testing.T, state tfsdk.State) WorkersBuildTriggerEnvironmentVariablesModel {
	t.Helper()
	var got WorkersBuildTriggerEnvironmentVariablesModel
	if diags := state.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get environment-variable state: %v", diags)
	}
	return got
}

func envHTTPTestDecodeVariables(t *testing.T, value types.Map) map[string]EnvironmentVariableModel {
	t.Helper()
	var got map[string]EnvironmentVariableModel
	if diags := value.ElementsAs(context.Background(), &got, false); diags.HasError() {
		t.Fatalf("decode environment-variable map: %v", diags)
	}
	return got
}

func TestWorkersBuildTriggerEnvironmentVariablesCreatePatchMapping(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.EscapedPath() != envHTTPTestPath() {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
		}
		switch req.Method {
		case http.MethodPatch:
			var body map[string]struct {
				Value    *string `json:"value"`
				IsSecret bool    `json:"is_secret"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Errorf("decode PATCH body: %v", err)
			}
			if body["API_KEY"].Value == nil || *body["API_KEY"].Value != envHTTPTestSecret || !body["API_KEY"].IsSecret {
				t.Errorf("secret variable was not mapped correctly: %#v", body["API_KEY"])
			}
			if body["NODE_ENV"].Value == nil || *body["NODE_ENV"].Value != "production" || body["NODE_ENV"].IsSecret {
				t.Errorf("plain variable was not mapped correctly: %#v", body["NODE_ENV"])
			}
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{}}`)
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, envHTTPTestEnvelope())
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	model := envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY":  {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
		"NODE_ENV": {Value: types.StringValue("production"), IsSecret: types.BoolValue(false)},
	})
	model.ID = types.StringNull()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	envHTTPTestResource(t, ts.URL).Create(ctx, resource.CreateRequest{Plan: envHTTPTestPlan(t, model)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", resp.Diagnostics)
	}
	got := envHTTPTestDecodeState(t, resp.State)
	variables := envHTTPTestDecodeVariables(t, got.Variables)
	if got.ID.ValueString() != envHTTPTestTriggerUUID || variables["API_KEY"].Value.ValueString() != envHTTPTestSecret {
		t.Fatalf("create did not reconcile redacted secret into state: %#v", variables)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesCreateDeletesUnexpectedRemoteVariables(t *testing.T) {
	ctx := context.Background()
	var methodsAndPaths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methodsAndPaths = append(methodsAndPaths, req.Method+" "+req.URL.EscapedPath())
		switch req.Method {
		case http.MethodPatch:
			// PATCH responses may be partial and are not authoritative for
			// complete-map ownership.
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{"NODE_ENV":{"is_secret":false,"value":"production"}}}`)
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{`+
				`"NODE_ENV":{"is_secret":false,"value":"production"},`+
				`"Z_EXTRA":{"is_secret":false,"value":"z"},`+
				`"A_EXTRA":{"is_secret":false,"value":"a"}`+
				`}}`)
		case http.MethodDelete:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[]}`)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}
	}))
	defer ts.Close()

	model := envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"NODE_ENV": {Value: types.StringValue("production"), IsSecret: types.BoolValue(false)},
	})
	model.ID = types.StringNull()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	envHTTPTestResource(t, ts.URL).Create(ctx, resource.CreateRequest{Plan: envHTTPTestPlan(t, model)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", resp.Diagnostics)
	}
	want := []string{
		http.MethodPatch + " " + envHTTPTestPath(),
		http.MethodGet + " " + envHTTPTestPath(),
		http.MethodDelete + " " + envHTTPTestPath() + "/A_EXTRA",
		http.MethodDelete + " " + envHTTPTestPath() + "/Z_EXTRA",
	}
	if fmt.Sprint(methodsAndPaths) != fmt.Sprint(want) {
		t.Fatalf("request order = %#v, want %#v", methodsAndPaths, want)
	}
	variables := envHTTPTestDecodeVariables(t, envHTTPTestDecodeState(t, resp.State).Variables)
	if len(variables) != 1 || variables["NODE_ENV"].Value.ValueString() != "production" {
		t.Fatalf("unexpected variables in final state: %#v", variables)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesCreateRetainsVerifiedStateWhenCleanupFails(t *testing.T) {
	ctx := context.Background()
	var methods []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		switch req.Method {
		case http.MethodPatch:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{}}`)
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{`+
				`"API_KEY":{"is_secret":true,"value":null},`+
				`"EXTRA":{"is_secret":false,"value":"remote-only"}`+
				`}}`)
		case http.MethodDelete:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":false,"errors":[{"code":9109,"message":"rejected `+envHTTPTestSecret+`"}]}`)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	model := envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	})
	model.ID = types.StringNull()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	envHTTPTestResource(t, ts.URL).Create(ctx, resource.CreateRequest{Plan: envHTTPTestPlan(t, model)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("cleanup failure must produce an error diagnostic")
	}
	if strings.Contains(fmt.Sprint(resp.Diagnostics), envHTTPTestSecret) {
		t.Fatalf("cleanup diagnostic leaked secret: %v", resp.Diagnostics)
	}
	if fmt.Sprint(methods) != fmt.Sprint([]string{http.MethodPatch, http.MethodGet, http.MethodDelete}) {
		t.Fatalf("request order = %#v, want PATCH then GET then DELETE", methods)
	}
	got := envHTTPTestDecodeState(t, resp.State)
	variables := envHTTPTestDecodeVariables(t, got.Variables)
	if got.ID.ValueString() != envHTTPTestTriggerUUID || len(variables) != 1 || variables["API_KEY"].Value.ValueString() != envHTTPTestSecret {
		t.Fatalf("cleanup failure lost verified desired state: id=%q variables=%#v", got.ID.ValueString(), variables)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesCreateRejectsMissingDesiredVariableAfterGET(t *testing.T) {
	ctx := context.Background()
	var methods []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		switch req.Method {
		case http.MethodPatch:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{}}`)
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{}}`)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	model := envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	})
	model.ID = types.StringNull()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	envHTTPTestResource(t, ts.URL).Create(ctx, resource.CreateRequest{Plan: envHTTPTestPlan(t, model)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("authoritative GET missing a desired variable must fail create")
	}
	if strings.Contains(fmt.Sprint(resp.Diagnostics), envHTTPTestSecret) {
		t.Fatalf("missing-variable diagnostic leaked secret: %v", resp.Diagnostics)
	}
	if fmt.Sprint(methods) != fmt.Sprint([]string{http.MethodPatch, http.MethodGet}) {
		t.Fatalf("request order = %#v, want PATCH then GET with no DELETE", methods)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatalf("failed create synthesized state: %s", resp.State.Raw)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesReadGETPreservesSecret(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.EscapedPath() != envHTTPTestPath() {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
		}
		envHTTPTestWriteJSON(t, w, http.StatusOK, envHTTPTestEnvelope())
	}))
	defer ts.Close()

	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY":  {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
		"NODE_ENV": {Value: types.StringValue("old"), IsSecret: types.BoolValue(false)},
	}))
	resp := &resource.ReadResponse{State: state}
	envHTTPTestResource(t, ts.URL).Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	variables := envHTTPTestDecodeVariables(t, envHTTPTestDecodeState(t, resp.State).Variables)
	if variables["API_KEY"].Value.ValueString() != envHTTPTestSecret {
		t.Fatalf("GET lost redacted secret: %#v", variables["API_KEY"])
	}
	if variables["NODE_ENV"].Value.ValueString() != "production" {
		t.Fatalf("GET did not reconcile plain value: %#v", variables["NODE_ENV"])
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesReadIncludesRemoteDrift(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{`+
			`"NODE_ENV":{"is_secret":false,"value":"production"},`+
			`"EXTRA":{"is_secret":false,"value":"remote-only"}`+
			`}}`)
	}))
	defer ts.Close()

	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"NODE_ENV": {Value: types.StringValue("production"), IsSecret: types.BoolValue(false)},
	}))
	resp := &resource.ReadResponse{State: state}
	envHTTPTestResource(t, ts.URL).Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	variables := envHTTPTestDecodeVariables(t, envHTTPTestDecodeState(t, resp.State).Variables)
	if len(variables) != 2 || variables["EXTRA"].Value.ValueString() != "remote-only" {
		t.Fatalf("remote drift was not hydrated into state: %#v", variables)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesUpdatePrunesAuthoritativeRemoteMapAfterPatch(t *testing.T) {
	ctx := context.Background()
	var methods []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		switch req.Method {
		case http.MethodPatch:
			if req.URL.EscapedPath() != envHTTPTestPath() {
				t.Errorf("PATCH path = %q", req.URL.EscapedPath())
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Errorf("decode PATCH body: %v", err)
			}
			if _, exists := body["OLD KEY"]; exists || len(body) != 1 {
				t.Errorf("PATCH must own only desired variables: %#v", body)
			}
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{}}`)
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{`+
				`"API_KEY":{"is_secret":true,"value":null},`+
				`"OLD KEY":{"is_secret":false,"value":"remote-only"}`+
				`}}`)
		case http.MethodDelete:
			want := envHTTPTestPath() + "/OLD%20KEY"
			if req.URL.EscapedPath() != want {
				t.Errorf("DELETE path = %q, want %q", req.URL.EscapedPath(), want)
			}
			envHTTPTestWriteJSON(t, w, http.StatusNotFound, `{"success":false,"errors":[{"code":1000,"message":"already absent"}]}`)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}
	}))
	defer ts.Close()

	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue("old-secret"), IsSecret: types.BoolValue(true)},
	}))
	plan := envHTTPTestPlan(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	}))
	resp := &resource.UpdateResponse{State: state}
	envHTTPTestResource(t, ts.URL).Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}
	if len(methods) != 3 || methods[0] != http.MethodPatch || methods[1] != http.MethodGet || methods[2] != http.MethodDelete {
		t.Fatalf("request order = %#v, want PATCH then GET then DELETE", methods)
	}
	variables := envHTTPTestDecodeVariables(t, envHTTPTestDecodeState(t, resp.State).Variables)
	if len(variables) != 1 || variables["API_KEY"].Value.ValueString() != envHTTPTestSecret {
		t.Fatalf("update did not preserve desired secret: %#v", variables)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesUpdateRejectsMissingDesiredVariableAfterGET(t *testing.T) {
	ctx := context.Background()
	var methods []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		switch req.Method {
		case http.MethodPatch:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{}}`)
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{}}`)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue("old-secret"), IsSecret: types.BoolValue(true)},
	}))
	plan := envHTTPTestPlan(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	}))
	resp := &resource.UpdateResponse{State: state}
	envHTTPTestResource(t, ts.URL).Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("authoritative GET missing a desired variable must fail update")
	}
	if fmt.Sprint(methods) != fmt.Sprint([]string{http.MethodPatch, http.MethodGet}) {
		t.Fatalf("request order = %#v, want PATCH then GET with no DELETE", methods)
	}
	variables := envHTTPTestDecodeVariables(t, envHTTPTestDecodeState(t, resp.State).Variables)
	if got := variables["API_KEY"].Value.ValueString(); got != "old-secret" {
		t.Fatalf("failed update advanced state to %q", got)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesUpdateStopsOnVariableConversionDiagnostics(t *testing.T) {
	ctx := context.Background()
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer ts.Close()

	resp := &resource.UpdateResponse{}
	envHTTPTestResource(t, ts.URL).Update(ctx, resource.UpdateRequest{Plan: envHTTPTestInvalidVariablesPlan(t)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("invalid environment-variable element must produce conversion diagnostics")
	}
	if requestCount != 0 {
		t.Fatalf("update made %d request(s) after conversion diagnostics", requestCount)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesDeleteUsesPerKeyDELETE(t *testing.T) {
	ctx := context.Background()
	var methodsAndPaths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methodsAndPaths = append(methodsAndPaths, req.Method+" "+req.URL.EscapedPath())
		switch req.Method {
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{`+
				`"Z_VAR":{"is_secret":false,"value":"z"},`+
				`"A_VAR":{"is_secret":false,"value":"a"}`+
				`}}`)
		case http.MethodDelete:
			if strings.HasSuffix(req.URL.EscapedPath(), "/A_VAR") {
				envHTTPTestWriteJSON(t, w, http.StatusNotFound, `{"success":false,"errors":[{"code":1000,"message":"already absent"}]}`)
				return
			}
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[]}`)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"STALE_STATE_ONLY": {Value: types.StringValue("stale"), IsSecret: types.BoolValue(false)},
	}))
	resp := &resource.DeleteResponse{}
	envHTTPTestResource(t, ts.URL).Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", resp.Diagnostics)
	}
	want := []string{
		http.MethodGet + " " + envHTTPTestPath(),
		http.MethodDelete + " " + envHTTPTestPath() + "/A_VAR",
		http.MethodDelete + " " + envHTTPTestPath() + "/Z_VAR",
	}
	if fmt.Sprint(methodsAndPaths) != fmt.Sprint(want) {
		t.Fatalf("requests = %#v, want %#v", methodsAndPaths, want)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesDeleteGET404IsIdempotent(t *testing.T) {
	ctx := context.Background()
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		if req.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", req.Method)
		}
		envHTTPTestWriteJSON(t, w, http.StatusNotFound, `{"success":false,"errors":[{"code":1000,"message":"not found"}],"result":{}}`)
	}))
	defer ts.Close()

	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	}))
	resp := &resource.DeleteResponse{}
	envHTTPTestResource(t, ts.URL).Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() || requestCount != 1 {
		t.Fatalf("GET 404 must complete delete idempotently: requests=%d diagnostics=%v", requestCount, resp.Diagnostics)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesSuccessFalseDoesNotLeakSecret(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", req.Method)
		}
		envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":false,"errors":[{"code":9109,"message":"rejected `+envHTTPTestSecret+`"}],"result":{}}`)
	}))
	defer ts.Close()

	model := envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	})
	model.ID = types.StringNull()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	envHTTPTestResource(t, ts.URL).Create(ctx, resource.CreateRequest{Plan: envHTTPTestPlan(t, model)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("success=false must produce an error diagnostic")
	}
	diagnosticText := fmt.Sprint(resp.Diagnostics)
	if strings.Contains(diagnosticText, envHTTPTestSecret) {
		t.Fatalf("diagnostic leaked request secret: %s", diagnosticText)
	}
	if !strings.Contains(diagnosticText, "9109") {
		t.Fatalf("diagnostic omitted Cloudflare envelope error code: %s", diagnosticText)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesGETSuccessFalseAnd404(t *testing.T) {
	ctx := context.Background()
	responseStatus := http.StatusOK
	responseBody := `{"success":false,"errors":[{"code":9109,"message":"Workers CI denied"}],"result":{}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		envHTTPTestWriteJSON(t, w, responseStatus, responseBody)
	}))
	defer ts.Close()

	r := envHTTPTestResource(t, ts.URL)
	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{}))
	errorResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, errorResp)
	if !errorResp.Diagnostics.HasError() {
		t.Fatal("GET success=false must produce an error diagnostic")
	}

	responseStatus = http.StatusNotFound
	responseBody = `{"success":false,"errors":[{"code":1000,"message":"not found"}],"result":{}}`
	notFoundResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, notFoundResp)
	if notFoundResp.Diagnostics.HasError() || !notFoundResp.State.Raw.IsNull() {
		t.Fatalf("GET 404 must remove state: state=%s diagnostics=%v", notFoundResp.State.Raw, notFoundResp.Diagnostics)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesDeleteSuccessFalse(t *testing.T) {
	ctx := context.Background()
	var methods []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		switch req.Method {
		case http.MethodGet:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{"API_KEY":{"is_secret":true,"value":null}}}`)
		case http.MethodDelete:
			envHTTPTestWriteJSON(t, w, http.StatusOK, `{"success":false,"errors":[{"code":9109,"message":"delete denied"}]}`)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	state := envHTTPTestState(t, envHTTPTestModel(t, map[string]EnvironmentVariableModel{
		"API_KEY": {Value: types.StringValue(envHTTPTestSecret), IsSecret: types.BoolValue(true)},
	}))
	resp := &resource.DeleteResponse{}
	envHTTPTestResource(t, ts.URL).Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("DELETE success=false must produce an error diagnostic")
	}
	if fmt.Sprint(methods) != fmt.Sprint([]string{http.MethodGet, http.MethodDelete}) {
		t.Fatalf("request order = %#v, want GET then DELETE", methods)
	}
	if strings.Contains(fmt.Sprint(resp.Diagnostics), envHTTPTestSecret) {
		t.Fatalf("DELETE diagnostic leaked secret: %v", resp.Diagnostics)
	}
}

func TestWorkersBuildTriggerEnvironmentVariablesImportAndReconciliation(t *testing.T) {
	ctx := context.Background()
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		if req.Method != http.MethodGet || req.URL.EscapedPath() != envHTTPTestPath() {
			t.Errorf("unexpected import request: %s %s", req.Method, req.URL.EscapedPath())
		}
		envHTTPTestWriteJSON(t, w, http.StatusOK, envHTTPTestEnvelope())
	}))
	defer ts.Close()

	r := envHTTPTestResource(t, ts.URL)
	invalidResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: envHTTPTestAccountID}, invalidResp)
	if !invalidResp.Diagnostics.HasError() || requestCount != 0 {
		t.Fatalf("invalid import should fail before HTTP: requests=%d diagnostics=%v", requestCount, invalidResp.Diagnostics)
	}

	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: envHTTPTestAccountID + "/" + envHTTPTestTriggerUUID}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %v", resp.Diagnostics)
	}
	got := envHTTPTestDecodeState(t, resp.State)
	if got.ID.ValueString() != envHTTPTestTriggerUUID || got.AccountID.ValueString() != envHTTPTestAccountID || got.TriggerUUID.ValueString() != envHTTPTestTriggerUUID {
		t.Fatalf("unexpected imported identity: %#v", got)
	}
	variables := envHTTPTestDecodeVariables(t, got.Variables)
	if !variables["API_KEY"].Value.IsNull() {
		t.Fatalf("imported redacted secret must remain null: %#v", variables["API_KEY"])
	}
	if variables["NODE_ENV"].Value.ValueString() != "production" {
		t.Fatalf("imported plain variable was not reconciled: %#v", variables["NODE_ENV"])
	}
	diagnosticText := fmt.Sprint(resp.Diagnostics)
	if !strings.Contains(diagnosticText, "secret environment variables must be reconciled") {
		t.Fatalf("import omitted the redacted-secret warning: %s", diagnosticText)
	}
	if strings.Contains(diagnosticText, envHTTPTestSecret) {
		t.Fatalf("import warning leaked a secret value: %s", diagnosticText)
	}
}
