package workers_build_trigger

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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	httpTestAccountID      = "023e105f4ecef8ad9ca31a8372d0c353"
	httpTestWorkerTag      = "72edf31f83e240448fce38bef56104e3"
	httpTestRepositoryUUID = "11111111-1111-4111-8111-111111111111"
	httpTestBuildTokenUUID = "22222222-2222-4222-8222-222222222222"
	httpTestTriggerUUID    = "33333333-3333-4333-8333-333333333333"
)

func TestRequestFromModel(t *testing.T) {
	ctx := context.Background()
	stringsSet := func(values ...string) types.Set {
		set, diags := types.SetValueFrom(ctx, types.StringType, values)
		if diags.HasError() {
			t.Fatalf("failed to make set: %v", diags)
		}
		return set
	}
	model := WorkersBuildTriggerModel{
		ExternalScriptID:     types.StringValue("72edf31f83e240448fce38bef56104e3"),
		RepositoryConnection: types.StringValue("11111111-1111-4111-8111-111111111111"),
		BuildTokenUUID:       types.StringValue("22222222-2222-4222-8222-222222222222"),
		TriggerName:          types.StringValue("preview"),
		BuildCommand:         types.StringValue("npm run cf:build"),
		DeployCommand:        types.StringValue("npm run cf:upload:built"),
		RootDirectory:        types.StringValue("/"),
		BranchIncludes:       stringsSet("feature/*", "*"),
		BranchExcludes:       stringsSet("main"),
		PathIncludes:         stringsSet("*"),
		PathExcludes:         stringsSet("*.md"),
		BuildCachingEnabled:  types.BoolValue(true),
	}
	request, diags := requestFromModel(ctx, model, true)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["external_script_id"] != model.ExternalScriptID.ValueString() {
		t.Fatalf("external_script_id was not serialized: %s", encoded)
	}
	if decoded["repo_connection_uuid"] != model.RepositoryConnection.ValueString() {
		t.Fatalf("repo_connection_uuid was not serialized: %s", encoded)
	}
	branches := request.BranchIncludes
	if len(branches) != 2 || branches[0] != "*" || branches[1] != "feature/*" {
		t.Fatalf("sets must serialize deterministically, got %#v", branches)
	}
}

func TestUpdateRequestOmitsImmutableFields(t *testing.T) {
	ctx := context.Background()
	empty := httpTestStringSet(t)
	model := WorkersBuildTriggerModel{
		ExternalScriptID:     types.StringValue("72edf31f83e240448fce38bef56104e3"),
		RepositoryConnection: types.StringValue("11111111-1111-4111-8111-111111111111"),
		BuildTokenUUID:       types.StringValue("22222222-2222-4222-8222-222222222222"),
		TriggerName:          types.StringValue("staging"),
		BuildCommand:         types.StringValue("build"),
		DeployCommand:        types.StringValue("deploy"),
		RootDirectory:        types.StringValue("/"),
		BranchIncludes:       empty,
		BranchExcludes:       empty,
		PathIncludes:         empty,
		PathExcludes:         empty,
		BuildCachingEnabled:  types.BoolValue(false),
	}
	request, diags := requestFromModel(ctx, model, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["external_script_id"]; ok {
		t.Fatalf("update serialized immutable Worker tag: %s", encoded)
	}
	if _, ok := decoded["repo_connection_uuid"]; ok {
		t.Fatalf("update serialized immutable repository connection: %s", encoded)
	}
}

func TestTriggerResponseEnvelope(t *testing.T) {
	payload := []byte(`{"success":true,"result":{"trigger_uuid":"33333333-3333-4333-8333-333333333333","external_script_id":"72edf31f83e240448fce38bef56104e3","build_token_uuid":"22222222-2222-4222-8222-222222222222","trigger_name":"staging","build_command":"npm run cf:build","deploy_command":"npm run cf:deploy:built","root_directory":"/","branch_includes":["main"],"branch_excludes":[],"path_includes":["*"],"path_excludes":[],"build_caching_enabled":true,"repo_connection":{"repo_connection_uuid":"11111111-1111-4111-8111-111111111111"}}}`)
	var envelope triggerEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || envelope.Result.TriggerUUID == nil || *envelope.Result.TriggerUUID != "33333333-3333-4333-8333-333333333333" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
	if envelope.Result.Repository == nil || envelope.Result.Repository.UUID == nil || *envelope.Result.Repository.UUID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("repository connection was not decoded: %#v", envelope.Result.Repository)
	}
}

func TestTriggerEnvelopeRejectsErrorsEvenWhenSuccessIsTrue(t *testing.T) {
	if err := validateEnvelope(true, []apiError{{Code: 9109, Message: "unexpected error"}}); err == nil {
		t.Fatal("an envelope containing errors must be rejected")
	}
}

func httpTestStringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("make string set: %v", diags)
	}
	return set
}

func httpTestTriggerModel(t *testing.T, name string) WorkersBuildTriggerModel {
	t.Helper()
	return WorkersBuildTriggerModel{
		ID:                   types.StringValue(httpTestTriggerUUID),
		AccountID:            types.StringValue(httpTestAccountID),
		ExternalScriptID:     types.StringValue(httpTestWorkerTag),
		RepositoryConnection: types.StringValue(httpTestRepositoryUUID),
		BuildTokenUUID:       types.StringValue(httpTestBuildTokenUUID),
		TriggerName:          types.StringValue(name),
		BuildCommand:         types.StringValue("npm run cf:build"),
		DeployCommand:        types.StringValue("npm run cf:deploy:built"),
		RootDirectory:        types.StringValue("/app"),
		BranchIncludes:       httpTestStringSet(t, "main"),
		BranchExcludes:       httpTestStringSet(t),
		PathIncludes:         httpTestStringSet(t, "src/**"),
		PathExcludes:         httpTestStringSet(t, "**/*.md"),
		BuildCachingEnabled:  types.BoolValue(true),
		CreatedOn:            types.StringValue("2026-08-15T00:00:00Z"),
		ModifiedOn:           types.StringValue("2026-08-15T00:01:00Z"),
	}
}

func httpTestTriggerState(t *testing.T, model WorkersBuildTriggerModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: ResourceSchema(context.Background())}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set trigger state: %v", diags)
	}
	return state
}

func httpTestTriggerPlan(t *testing.T, model WorkersBuildTriggerModel) tfsdk.Plan {
	t.Helper()
	state := httpTestTriggerState(t, model)
	return tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
}

func httpTestTriggerResource(t *testing.T, baseURL string) *WorkersBuildTriggerResource {
	t.Helper()
	r := NewResource().(*WorkersBuildTriggerResource)
	client := cloudflare.NewClient(option.WithBaseURL(baseURL), option.WithAPIToken("test-token"))
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("configure trigger resource: %v", resp.Diagnostics)
	}
	return r
}

func httpTestTriggerResult(name string) string {
	return fmt.Sprintf(`{
		"trigger_uuid":%q,
		"external_script_id":%q,
		"build_token_uuid":%q,
		"trigger_name":%q,
		"build_command":"npm run cf:build",
		"deploy_command":"npm run cf:deploy:built",
		"root_directory":"/app",
		"branch_includes":["main"],
		"branch_excludes":[],
		"path_includes":["src/**"],
		"path_excludes":["**/*.md"],
		"build_caching_enabled":true,
		"created_on":"2026-08-15T00:00:00Z",
		"modified_on":"2026-08-15T00:01:00Z",
		"repo_connection":{"repo_connection_uuid":%q}
	}`, httpTestTriggerUUID, httpTestWorkerTag, httpTestBuildTokenUUID, name, httpTestRepositoryUUID)
}

func httpTestWriteJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func httpTestAssertTriggerState(t *testing.T, state tfsdk.State, name string) {
	t.Helper()
	var got WorkersBuildTriggerModel
	if diags := state.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get trigger state: %v", diags)
	}
	if got.ID.ValueString() != httpTestTriggerUUID || got.ExternalScriptID.ValueString() != httpTestWorkerTag {
		t.Fatalf("unexpected trigger identity: id=%q worker=%q", got.ID.ValueString(), got.ExternalScriptID.ValueString())
	}
	if got.RepositoryConnection.ValueString() != httpTestRepositoryUUID {
		t.Fatalf("repository connection = %q", got.RepositoryConnection.ValueString())
	}
	if got.TriggerName.ValueString() != name {
		t.Fatalf("trigger_name = %q, want %q", got.TriggerName.ValueString(), name)
	}
}

func TestWorkersBuildTriggerCreateHTTPMapping(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/accounts/"+httpTestAccountID+"/builds/triggers" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("decode create body: %v", err)
		}
		if body["external_script_id"] != httpTestWorkerTag || body["repo_connection_uuid"] != httpTestRepositoryUUID {
			t.Errorf("create omitted immutable fields: %#v", body)
		}
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{"trigger_uuid":"`+httpTestTriggerUUID+`"}}`)
	}))
	defer ts.Close()

	model := httpTestTriggerModel(t, "staging")
	model.ID = types.StringNull()
	model.CreatedOn = types.StringUnknown()
	model.ModifiedOn = types.StringUnknown()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	httpTestTriggerResource(t, ts.URL).Create(ctx, resource.CreateRequest{Plan: httpTestTriggerPlan(t, model)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", resp.Diagnostics)
	}
	httpTestAssertTriggerState(t, resp.State, "staging")
	var got WorkersBuildTriggerModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get created state: %v", diags)
	}
	if !got.CreatedOn.IsNull() || !got.ModifiedOn.IsNull() {
		t.Fatalf("omitted computed timestamps must become null, got created=%s modified=%s", got.CreatedOn, got.ModifiedOn)
	}
}

func TestWorkersBuildTriggerReadUsesWorkerListEndpoint(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		wantPath := "/accounts/" + httpTestAccountID + "/builds/workers/" + httpTestWorkerTag + "/triggers"
		if req.Method != http.MethodGet || req.URL.EscapedPath() != wantPath {
			t.Errorf("unexpected request: %s %s, want GET %s", req.Method, req.URL.EscapedPath(), wantPath)
		}
		deleted := strings.Replace(httpTestTriggerResult("deleted"), httpTestTriggerUUID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 1)
		deleted = strings.TrimSuffix(deleted, "}") + `,"deleted_on":"2026-08-15T00:02:00Z"}`
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":[`+deleted+`,`+httpTestTriggerResult("from-api")+`]}`)
	}))
	defer ts.Close()

	state := httpTestTriggerState(t, httpTestTriggerModel(t, "from-state"))
	resp := &resource.ReadResponse{State: state}
	httpTestTriggerResource(t, ts.URL).Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	httpTestAssertTriggerState(t, resp.State, "from-api")
}

func TestWorkersBuildTriggerReadRemovesDeletedTriggerWithoutPaginationMetadata(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		deleted := strings.TrimSuffix(httpTestTriggerResult("deleted"), "}") + `,"deleted_on":"2026-08-15T00:02:00Z"}`
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":[`+deleted+`]}`)
	}))
	defer ts.Close()

	state := httpTestTriggerState(t, httpTestTriggerModel(t, "from-state"))
	resp := &resource.ReadResponse{State: state}
	httpTestTriggerResource(t, ts.URL).Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("a matching soft-deleted trigger must be removed from state even without pagination metadata")
	}
}

func TestWorkersBuildTriggerReadFindsTriggerOnLaterPage(t *testing.T) {
	ctx := context.Background()
	requests := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		wantPath := "/accounts/" + httpTestAccountID + "/builds/workers/" + httpTestWorkerTag + "/triggers"
		if req.Method != http.MethodGet || req.URL.EscapedPath() != wantPath {
			t.Errorf("unexpected request: %s %s, want GET %s", req.Method, req.URL.EscapedPath(), wantPath)
		}
		switch req.URL.Query().Get("page") {
		case "1":
			other := strings.Replace(httpTestTriggerResult("other"), httpTestTriggerUUID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 1)
			httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":[`+other+`],"result_info":{"page":1,"total_pages":2}}`)
		case "2":
			httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":[`+httpTestTriggerResult("from-page-two")+`],"result_info":{"page":2,"total_pages":2}}`)
		default:
			t.Errorf("unexpected page %q", req.URL.Query().Get("page"))
			httpTestWriteJSON(t, w, http.StatusBadRequest, `{"success":false,"errors":[{"code":1000,"message":"unexpected page"}]}`)
			return
		}
	}))
	defer ts.Close()

	state := httpTestTriggerState(t, httpTestTriggerModel(t, "from-state"))
	resp := &resource.ReadResponse{State: state}
	httpTestTriggerResource(t, ts.URL).Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	httpTestAssertTriggerState(t, resp.State, "from-page-two")
}

func TestWorkersBuildTriggerReadPreservesStateWithoutPaginationMetadata(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name       string
		resultInfo string
	}{
		{name: "result_info omitted"},
		{name: "total_pages omitted", resultInfo: `,"result_info":{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				other := strings.Replace(httpTestTriggerResult("other"), httpTestTriggerUUID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 1)
				httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":[`+other+`]`+test.resultInfo+`}`)
			}))
			defer ts.Close()

			state := httpTestTriggerState(t, httpTestTriggerModel(t, "from-state"))
			resp := &resource.ReadResponse{State: state}
			httpTestTriggerResource(t, ts.URL).Read(ctx, resource.ReadRequest{State: state}, resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("missing pagination metadata must be an error when the managed trigger was not found")
			}
			if resp.State.Raw.IsNull() {
				t.Fatal("ambiguous pagination response must preserve Terraform state")
			}
		})
	}
}

func TestWorkersBuildTriggerUpdateHTTPMapping(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		wantPath := "/accounts/" + httpTestAccountID + "/builds/triggers/" + httpTestTriggerUUID
		if req.Method != http.MethodPatch || req.URL.EscapedPath() != wantPath {
			t.Errorf("unexpected request: %s %s, want PATCH %s", req.Method, req.URL.EscapedPath(), wantPath)
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("decode update body: %v", err)
		}
		if _, exists := body["external_script_id"]; exists {
			t.Errorf("update sent immutable external_script_id: %#v", body)
		}
		if _, exists := body["repo_connection_uuid"]; exists {
			t.Errorf("update sent immutable repo_connection_uuid: %#v", body)
		}
		// PATCH fields are optional in Cloudflare's response schema. A partial
		// response must not erase immutable identity or other planned values.
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{"trigger_name":"updated","modified_on":"2026-08-15T00:02:00Z"}}`)
	}))
	defer ts.Close()

	state := httpTestTriggerState(t, httpTestTriggerModel(t, "staging"))
	planModel := httpTestTriggerModel(t, "updated")
	planModel.CreatedOn = types.StringUnknown()
	planModel.ModifiedOn = types.StringUnknown()
	plan := httpTestTriggerPlan(t, planModel)
	resp := &resource.UpdateResponse{State: state}
	httpTestTriggerResource(t, ts.URL).Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}
	httpTestAssertTriggerState(t, resp.State, "updated")
	var got WorkersBuildTriggerModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get updated state: %v", diags)
	}
	if got.CreatedOn.ValueString() != "2026-08-15T00:00:00Z" || got.ModifiedOn.ValueString() != "2026-08-15T00:02:00Z" {
		t.Fatalf("partial update response erased timestamps: created=%q modified=%q", got.CreatedOn.ValueString(), got.ModifiedOn.ValueString())
	}
}

func TestWorkersBuildTriggerCreateRequiresReturnedUUID(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":{"trigger_name":"staging"}}`)
	}))
	defer ts.Close()

	model := httpTestTriggerModel(t, "staging")
	model.ID = types.StringNull()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	httpTestTriggerResource(t, ts.URL).Create(ctx, resource.CreateRequest{Plan: httpTestTriggerPlan(t, model)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("create must reject a response without a trigger UUID")
	}
}

func TestWorkersBuildTriggerDeleteHTTPMapping(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		wantPath := "/accounts/" + httpTestAccountID + "/builds/triggers/" + httpTestTriggerUUID
		if req.Method != http.MethodDelete || req.URL.EscapedPath() != wantPath {
			t.Errorf("unexpected request: %s %s, want DELETE %s", req.Method, req.URL.EscapedPath(), wantPath)
		}
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[]}`)
	}))
	defer ts.Close()

	state := httpTestTriggerState(t, httpTestTriggerModel(t, "staging"))
	resp := &resource.DeleteResponse{}
	httpTestTriggerResource(t, ts.URL).Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", resp.Diagnostics)
	}
}

func TestWorkersBuildTriggerRead404RemovesStateAndDelete404Succeeds(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpTestWriteJSON(t, w, http.StatusNotFound, `{"success":false,"errors":[{"code":1000,"message":"not found"}]}`)
	}))
	defer ts.Close()

	r := httpTestTriggerResource(t, ts.URL)
	state := httpTestTriggerState(t, httpTestTriggerModel(t, "staging"))
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Fatalf("404 read must remove state without error: state=%s diags=%v", readResp.State.Raw, readResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("404 delete diagnostics: %v", deleteResp.Diagnostics)
	}
}

func TestWorkersBuildTriggerSuccessFalseIsDiagnostic(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":false,"errors":[{"code":9109,"message":"Workers CI denied"}],"result":[]}`)
	}))
	defer ts.Close()

	state := httpTestTriggerState(t, httpTestTriggerModel(t, "staging"))
	resp := &resource.ReadResponse{State: state}
	httpTestTriggerResource(t, ts.URL).Read(ctx, resource.ReadRequest{State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("success=false must produce an error diagnostic")
	}
	if got := fmt.Sprint(resp.Diagnostics); !containsAll(got, "9109", "Workers CI denied") {
		t.Fatalf("diagnostic did not preserve Cloudflare envelope error: %s", got)
	}
}

func TestWorkersBuildTriggerImportHTTPAndParse(t *testing.T) {
	ctx := context.Background()
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		wantPath := "/accounts/" + httpTestAccountID + "/builds/workers/" + httpTestWorkerTag + "/triggers"
		if req.Method != http.MethodGet || req.URL.EscapedPath() != wantPath {
			t.Errorf("unexpected import request: %s %s", req.Method, req.URL.EscapedPath())
		}
		httpTestWriteJSON(t, w, http.StatusOK, `{"success":true,"errors":[],"result":[`+httpTestTriggerResult("imported")+`]}`)
	}))
	defer ts.Close()

	r := httpTestTriggerResource(t, ts.URL)
	invalidResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: httpTestAccountID + "/" + httpTestWorkerTag}, invalidResp)
	if !invalidResp.Diagnostics.HasError() || requestCount != 0 {
		t.Fatalf("invalid import should fail before HTTP: requests=%d diagnostics=%v", requestCount, invalidResp.Diagnostics)
	}

	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: ResourceSchema(ctx)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: httpTestAccountID + "/" + httpTestWorkerTag + "/" + httpTestTriggerUUID}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %v", resp.Diagnostics)
	}
	httpTestAssertTriggerState(t, resp.State, "imported")
}

func containsAll(value string, substrings ...string) bool {
	for _, substring := range substrings {
		if !strings.Contains(value, substring) {
			return false
		}
	}
	return true
}
