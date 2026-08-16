package workers_subdomain_test

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
	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_subdomain"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testAccountID = "023e105f4ecef8ad9ca31a8372d0c353"
	testToken     = "unit-test-token-must-not-leak"
)

func newWorkersSubdomainResource(t *testing.T, baseURL string) *workers_subdomain.WorkersSubdomainResource {
	t.Helper()

	r := workers_subdomain.NewResource().(*workers_subdomain.WorkersSubdomainResource)
	client := cloudflare.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIToken(testToken),
	)
	configureResp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("configure resource: %v", configureResp.Diagnostics)
	}
	return r
}

func resourceState(t *testing.T, model workers_subdomain.WorkersSubdomainModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: workers_subdomain.ResourceSchema(context.Background())}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set resource state: %v", diags)
	}
	return state
}

func resourcePlan(t *testing.T, model workers_subdomain.WorkersSubdomainModel) tfsdk.Plan {
	t.Helper()
	state := resourceState(t, model)
	return tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
}

func workersSubdomainEnvelope(subdomain string) string {
	return fmt.Sprintf(`{"success":true,"errors":[],"messages":[],"result":{"subdomain":%q}}`, subdomain)
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func assertRequest(t *testing.T, req *http.Request, method string) {
	t.Helper()
	if req.Method != method {
		t.Errorf("method = %s, want %s", req.Method, method)
	}
	wantPath := "/accounts/" + testAccountID + "/workers/subdomain"
	if req.URL.Path != wantPath {
		t.Errorf("path = %q, want %q", req.URL.Path, wantPath)
	}
}

func assertPutBody(t *testing.T, req *http.Request, wantSubdomain string) bool {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Errorf("decode PUT body: %v", err)
		return false
	}
	if len(body) != 1 {
		t.Errorf("PUT body = %#v, want only the account-wide subdomain", body)
		return false
	}
	if got := body["subdomain"]; got != wantSubdomain {
		t.Errorf("subdomain = %#v, want %q", got, wantSubdomain)
	}
	if _, exists := body["enabled"]; exists {
		t.Error("account subdomain request must not contain per-script enabled")
	}
	if _, exists := body["previews_enabled"]; exists {
		t.Error("account subdomain request must not contain per-script previews_enabled")
	}
	return true
}

func TestWorkersSubdomainResourceCreate(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assertRequest(t, req, http.MethodPut)
		if !assertPutBody(t, req, "cc-games") {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		writeJSON(t, w, http.StatusOK, workersSubdomainEnvelope("cc-games"))
	}))
	defer ts.Close()

	r := newWorkersSubdomainResource(t, ts.URL)
	plan := resourcePlan(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringNull(),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("cc-games"),
	})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: workers_subdomain.ResourceSchema(ctx)}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", resp.Diagnostics)
	}

	assertResourceState(t, resp.State, "cc-games")
}

func TestWorkersSubdomainResourceUpdate(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assertRequest(t, req, http.MethodPut)
		if !assertPutBody(t, req, "new-label") {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		writeJSON(t, w, http.StatusOK, workersSubdomainEnvelope("new-label"))
	}))
	defer ts.Close()

	r := newWorkersSubdomainResource(t, ts.URL)
	state := resourceState(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("old-label"),
	})
	plan := resourcePlan(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("new-label"),
	})
	resp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}

	assertResourceState(t, resp.State, "new-label")
}

func TestWorkersSubdomainResourceRead(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assertRequest(t, req, http.MethodGet)
		writeJSON(t, w, http.StatusOK, workersSubdomainEnvelope("from-api"))
	}))
	defer ts.Close()

	r := newWorkersSubdomainResource(t, ts.URL)
	state := resourceState(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("from-state"),
	})
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	assertResourceState(t, resp.State, "from-api")
}

func TestWorkersSubdomainResourceRead404RemovesState(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assertRequest(t, req, http.MethodGet)
		writeJSON(t, w, http.StatusNotFound, `{"success":false,"errors":[{"code":1000,"message":"not found"}],"messages":[],"result":null}`)
	}))
	defer ts.Close()

	r := newWorkersSubdomainResource(t, ts.URL)
	state := resourceState(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("gone"),
	})
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("404 read diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatalf("state was not removed: %s", resp.State.Raw)
	}
}

func TestWorkersSubdomainResourceDelete(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assertRequest(t, req, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	r := newWorkersSubdomainResource(t, ts.URL)
	state := resourceState(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("cc-games"),
	})
	resp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", resp.Diagnostics)
	}
}

func TestWorkersSubdomainResourceImportByAccountID(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assertRequest(t, req, http.MethodGet)
		writeJSON(t, w, http.StatusOK, workersSubdomainEnvelope("imported-label"))
	}))
	defer ts.Close()

	r := newWorkersSubdomainResource(t, ts.URL)
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: workers_subdomain.ResourceSchema(ctx)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testAccountID}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %v", resp.Diagnostics)
	}

	assertResourceState(t, resp.State, "imported-label")
}

func TestWorkersSubdomainResourceErrorDoesNotExposeAuthorization(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusInternalServerError, `{"success":false,"errors":[{"code":1000,"message":"failure"}],"messages":[],"result":null}`)
	}))
	defer ts.Close()

	r := newWorkersSubdomainResource(t, ts.URL)
	state := resourceState(t, workers_subdomain.WorkersSubdomainModel{
		ID:        types.StringValue(testAccountID),
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringValue("cc-games"),
	})
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	diagnosticText := fmt.Sprint(resp.Diagnostics)
	if strings.Contains(diagnosticText, testToken) || strings.Contains(strings.ToLower(diagnosticText), "authorization") {
		t.Fatalf("diagnostic exposed an authorization header or token: %s", diagnosticText)
	}
}

func assertResourceState(t *testing.T, state tfsdk.State, wantSubdomain string) {
	t.Helper()
	var got workers_subdomain.WorkersSubdomainModel
	if diags := state.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get resource state: %v", diags)
	}
	if got.ID.ValueString() != testAccountID {
		t.Errorf("id = %q, want %q", got.ID.ValueString(), testAccountID)
	}
	if got.AccountID.ValueString() != testAccountID {
		t.Errorf("account_id = %q, want %q", got.AccountID.ValueString(), testAccountID)
	}
	if got.Subdomain.ValueString() != wantSubdomain {
		t.Errorf("subdomain = %q, want %q", got.Subdomain.ValueString(), wantSubdomain)
	}
}
