package workers_build_token_test

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
	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_build_token"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	tokenTestAccountID = "12ea67fb7ced068de03f35c22688e436"
	tokenTestUUID      = "182bd5e5-6e1a-4fe4-a799-aa6d9a6ab26e"
	tokenTestSecret    = "secret-value-that-must-never-appear-in-diagnostics"
	tokenTestCFID      = "ed17574386854bf78a67040be0a770b0"
)

func newBuildTokenResource(t *testing.T, baseURL string) *workers_build_token.WorkersBuildTokenResource {
	t.Helper()
	instance := workers_build_token.NewResource().(*workers_build_token.WorkersBuildTokenResource)
	client := cloudflare.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIToken("control-plane-test-token"),
	)
	response := &resource.ConfigureResponse{}
	instance.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("configure resource: %v", response.Diagnostics)
	}
	return instance
}

func tokenModel(id string, secret types.String) workers_build_token.WorkersBuildTokenModel {
	idValue := types.StringUnknown()
	if id != "" {
		idValue = types.StringValue(id)
	}
	return workers_build_token.WorkersBuildTokenModel{
		ID:                idValue,
		AccountID:         types.StringValue(tokenTestAccountID),
		BuildTokenName:    types.StringValue("terraform-deployment-token"),
		BuildTokenSecret:  secret,
		CloudflareTokenID: types.StringValue(tokenTestCFID),
		OwnerType:         types.StringValue("user"),
	}
}

func tokenPlan(t *testing.T, ctx context.Context, secret types.String) tfsdk.Plan {
	t.Helper()
	model := tokenModel("", secret)
	model.OwnerType = types.StringUnknown()
	plan := tfsdk.Plan{Schema: workers_build_token.ResourceSchema(ctx)}
	if diagnostics := plan.Set(ctx, &model); diagnostics.HasError() {
		t.Fatalf("set plan: %v", diagnostics)
	}
	return plan
}

func tokenState(t *testing.T, ctx context.Context, secret types.String) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: workers_build_token.ResourceSchema(ctx)}
	model := tokenModel(tokenTestUUID, secret)
	if diagnostics := state.Set(ctx, &model); diagnostics.HasError() {
		t.Fatalf("set state: %v", diagnostics)
	}
	return state
}

func tokenResponse(uuid, name, tokenID, owner string) string {
	return fmt.Sprintf(`{"build_token_name":%q,"build_token_uuid":%q,"cloudflare_token_id":%q,"owner_type":%q}`, name, uuid, tokenID, owner)
}

func diagnosticText(diagnostics diag.Diagnostics) string {
	var parts []string
	for _, diagnostic := range diagnostics {
		parts = append(parts, diagnostic.Summary(), diagnostic.Detail())
	}
	return strings.Join(parts, "\n")
}

func TestBuildTokenCreateMapsRequestPreservesSecretAndDecodesEnvelope(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/accounts/"+tokenTestAccountID+"/builds/tokens" {
			t.Errorf("path = %s", request.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["build_token_name"] != "terraform-deployment-token" || body["build_token_secret"] != tokenTestSecret || body["cloudflare_token_id"] != tokenTestCFID {
			t.Errorf("unexpected request body keys or values")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"result":%s}`, tokenResponse(tokenTestUUID, "terraform-deployment-token", tokenTestCFID, "user"))
	}))
	defer server.Close()

	response := &resource.CreateResponse{State: tfsdk.State{Schema: workers_build_token.ResourceSchema(ctx)}}
	newBuildTokenResource(t, server.URL).Create(ctx, resource.CreateRequest{Plan: tokenPlan(t, ctx, types.StringValue(tokenTestSecret))}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", response.Diagnostics)
	}

	var result workers_build_token.WorkersBuildTokenModel
	response.Diagnostics.Append(response.State.Get(ctx, &result)...)
	if result.ID.ValueString() != tokenTestUUID || result.BuildTokenSecret.ValueString() != tokenTestSecret {
		t.Fatal("create did not decode the ID and preserve the sensitive state value")
	}
}

func TestBuildTokenCreateRejectsNonUserOwnerAndCleansUp(t *testing.T) {
	ctx := context.Background()
	for _, owner := range []string{"", "account"} {
		t.Run("owner_"+owner, func(t *testing.T) {
			deleteCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch request.Method {
				case http.MethodPost:
					fmt.Fprintf(w, `{"success":true,"result":%s}`, tokenResponse(tokenTestUUID, "terraform-deployment-token", tokenTestCFID, owner))
				case http.MethodDelete:
					deleteCount++
					if request.URL.Path != "/accounts/"+tokenTestAccountID+"/builds/tokens/"+tokenTestUUID {
						t.Errorf("cleanup path = %s", request.URL.Path)
					}
					fmt.Fprint(w, `{"success":true,"errors":[],"result":null}`)
				default:
					t.Errorf("unexpected method %s", request.Method)
				}
			}))
			defer server.Close()

			response := &resource.CreateResponse{State: tfsdk.State{Schema: workers_build_token.ResourceSchema(ctx)}}
			newBuildTokenResource(t, server.URL).Create(ctx, resource.CreateRequest{Plan: tokenPlan(t, ctx, types.StringValue(tokenTestSecret))}, response)
			if !response.Diagnostics.HasError() {
				t.Fatal("non-user owner must be rejected")
			}
			if deleteCount != 1 {
				t.Fatalf("automatic cleanup count = %d, want 1", deleteCount)
			}
		})
	}
}

func TestBuildTokenCreateNeverIncludesSecretOrAuthorizationInDiagnostics(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(request.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"success":false,"errors":[{"message":"bad secret %s; Authorization: %s"}]}`, body["build_token_secret"], request.Header.Get("Authorization"))
	}))
	defer server.Close()

	response := &resource.CreateResponse{State: tfsdk.State{Schema: workers_build_token.ResourceSchema(ctx)}}
	newBuildTokenResource(t, server.URL).Create(ctx, resource.CreateRequest{Plan: tokenPlan(t, ctx, types.StringValue(tokenTestSecret))}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("expected create error")
	}
	diagnostics := diagnosticText(response.Diagnostics)
	if strings.Contains(diagnostics, tokenTestSecret) || strings.Contains(strings.ToLower(diagnostics), "authorization:") || strings.Contains(diagnostics, "control-plane-test-token") {
		t.Fatalf("sensitive value leaked in diagnostics: %s", diagnostics)
	}
}

func TestBuildTokenReadPaginatesFiltersUUIDAndPreservesSecret(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/accounts/"+tokenTestAccountID+"/builds/tokens" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.URL.Query().Get("per_page") != "200" {
			t.Errorf("per_page = %q", request.URL.Query().Get("per_page"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page") {
		case "1":
			fmt.Fprintf(w, `{"success":true,"result":[%s],"result_info":{"page":1,"total_pages":2}}`, tokenResponse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "other", "other-id", "user"))
		case "2":
			fmt.Fprintf(w, `{"success":true,"result":[%s],"result_info":{"page":2,"total_pages":2}}`, tokenResponse(tokenTestUUID, "renamed-remotely", tokenTestCFID, "user"))
		default:
			t.Fatalf("unexpected page %q", request.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	state := tokenState(t, ctx, types.StringValue(tokenTestSecret))
	response := &resource.ReadResponse{State: state}
	newBuildTokenResource(t, server.URL).Read(ctx, resource.ReadRequest{State: state}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", response.Diagnostics)
	}

	var result workers_build_token.WorkersBuildTokenModel
	response.Diagnostics.Append(response.State.Get(ctx, &result)...)
	if result.BuildTokenName.ValueString() != "renamed-remotely" {
		t.Errorf("name = %q", result.BuildTokenName.ValueString())
	}
	if result.BuildTokenSecret.ValueString() != tokenTestSecret {
		t.Errorf("secret was not preserved")
	}
}

func TestBuildTokenReadPaginatesWithoutResultInfo(t *testing.T) {
	ctx := context.Background()
	firstPage := make([]string, 200)
	for index := range firstPage {
		firstPage[index] = tokenResponse(fmt.Sprintf("other-%03d", index), "other", "other-id", "user")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page") {
		case "1":
			fmt.Fprintf(w, `{"success":true,"result":[%s]}`, strings.Join(firstPage, ","))
		case "2":
			fmt.Fprintf(w, `{"success":true,"result":[%s]}`, tokenResponse(tokenTestUUID, "from-page-two", tokenTestCFID, "user"))
		default:
			t.Fatalf("unexpected page %q", request.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	state := tokenState(t, ctx, types.StringValue(tokenTestSecret))
	response := &resource.ReadResponse{State: state}
	newBuildTokenResource(t, server.URL).Read(ctx, resource.ReadRequest{State: state}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", response.Diagnostics)
	}
	var result workers_build_token.WorkersBuildTokenModel
	response.Diagnostics.Append(response.State.Get(ctx, &result)...)
	if result.BuildTokenName.ValueString() != "from-page-two" {
		t.Fatalf("token from second page was not found: %#v", result)
	}
}

func TestBuildTokenReadStopsAtPaginationLimit(t *testing.T) {
	ctx := context.Background()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"result":[],"result_info":{"page":%s,"total_pages":101}}`, request.URL.Query().Get("page"))
	}))
	defer server.Close()

	state := tokenState(t, ctx, types.StringValue(tokenTestSecret))
	response := &resource.ReadResponse{State: state}
	newBuildTokenResource(t, server.URL).Read(ctx, resource.ReadRequest{State: state}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("pagination beyond the traversal limit must fail")
	}
	if requests != 100 {
		t.Fatalf("requests = %d, want 100", requests)
	}
	if got := diagnosticText(response.Diagnostics); !strings.Contains(got, "more Workers Builds token pages") {
		t.Fatalf("pagination diagnostic = %q", got)
	}
}

func TestBuildTokenReadCollection404KeepsStateAndErrors(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"success":false,"errors":[{"message":"account or endpoint unavailable"}]}`)
	}))
	defer server.Close()

	state := tokenState(t, ctx, types.StringValue(tokenTestSecret))
	response := &resource.ReadResponse{State: state}
	newBuildTokenResource(t, server.URL).Read(ctx, resource.ReadRequest{State: state}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("collection 404 must be an error, not proof that one token is absent")
	}
	if response.State.Raw.IsNull() {
		t.Fatal("collection 404 must preserve resource state")
	}
}

func TestBuildTokenReadRemovesMissingTokenFromState(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"result":[],"result_info":{"page":1,"total_pages":1}}`)
	}))
	defer server.Close()

	state := tokenState(t, ctx, types.StringValue(tokenTestSecret))
	response := &resource.ReadResponse{State: state}
	newBuildTokenResource(t, server.URL).Read(ctx, resource.ReadRequest{State: state}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", response.Diagnostics)
	}
	if !response.State.Raw.IsNull() {
		t.Fatal("missing token must be removed from state")
	}
}

func TestBuildTokenImportHydratesMetadataAndRequiresSecretReconciliation(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"result":[%s],"result_info":{"page":1,"total_pages":1}}`, tokenResponse(tokenTestUUID, "terraform-deployment-token", tokenTestCFID, "user"))
	}))
	defer server.Close()

	response := &resource.ImportStateResponse{State: tfsdk.State{Schema: workers_build_token.ResourceSchema(ctx)}}
	newBuildTokenResource(t, server.URL).ImportState(ctx, resource.ImportStateRequest{ID: tokenTestAccountID + "/" + tokenTestUUID}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %v", response.Diagnostics)
	}
	if len(response.Diagnostics.Warnings()) == 0 {
		t.Fatal("import must warn that the secret requires reconciliation")
	}

	var result workers_build_token.WorkersBuildTokenModel
	response.Diagnostics.Append(response.State.Get(ctx, &result)...)
	if result.ID.ValueString() != tokenTestUUID || result.BuildTokenName.ValueString() != "terraform-deployment-token" {
		t.Fatalf("import did not hydrate metadata: %#v", result)
	}
	if !result.BuildTokenSecret.IsNull() {
		t.Fatal("imported secret must be null because Cloudflare does not return it")
	}
}

func TestBuildTokenImportRejectsIncompleteMetadata(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		tokenName string
		tokenID   string
	}{
		{name: "missing token name", tokenID: tokenTestCFID},
		{name: "missing Cloudflare token ID", tokenName: "terraform-deployment-token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"success":true,"result":[%s],"result_info":{"page":1,"total_pages":1}}`, tokenResponse(tokenTestUUID, test.tokenName, test.tokenID, "user"))
			}))
			defer server.Close()

			response := &resource.ImportStateResponse{State: tfsdk.State{Schema: workers_build_token.ResourceSchema(ctx)}}
			newBuildTokenResource(t, server.URL).ImportState(ctx, resource.ImportStateRequest{ID: tokenTestAccountID + "/" + tokenTestUUID}, response)
			if !response.Diagnostics.HasError() {
				t.Fatal("incomplete import metadata must be rejected")
			}
		})
	}
}

func TestBuildTokenImportedSecretReconciliationIsLocalOnly(t *testing.T) {
	ctx := context.Background()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	state := tokenState(t, ctx, types.StringNull())
	planModel := tokenModel(tokenTestUUID, types.StringValue(tokenTestSecret))
	plan := tfsdk.Plan{Schema: workers_build_token.ResourceSchema(ctx)}
	if diagnostics := plan.Set(ctx, &planModel); diagnostics.HasError() {
		t.Fatalf("set plan: %v", diagnostics)
	}
	response := &resource.UpdateResponse{State: state}
	newBuildTokenResource(t, server.URL).Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", response.Diagnostics)
	}
	if requests != 0 {
		t.Fatalf("secret reconciliation made %d remote requests", requests)
	}

	var result workers_build_token.WorkersBuildTokenModel
	response.Diagnostics.Append(response.State.Get(ctx, &result)...)
	if result.BuildTokenSecret.ValueString() != tokenTestSecret {
		t.Fatal("configured secret was not recorded")
	}
}

func TestBuildTokenRejectsUnexpectedUpdates(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unexpected remote request")
	}))
	defer server.Close()

	tests := []struct {
		name  string
		state workers_build_token.WorkersBuildTokenModel
		plan  workers_build_token.WorkersBuildTokenModel
	}{
		{
			name:  "established secret",
			state: tokenModel(tokenTestUUID, types.StringValue(tokenTestSecret)),
			plan:  tokenModel(tokenTestUUID, types.StringValue(tokenTestSecret)),
		},
		{
			name:  "immutable field changed during import reconciliation",
			state: tokenModel(tokenTestUUID, types.StringNull()),
			plan:  tokenModel(tokenTestUUID, types.StringValue(tokenTestSecret)),
		},
	}
	tests[1].plan.BuildTokenName = types.StringValue("changed-name")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := tfsdk.State{Schema: workers_build_token.ResourceSchema(ctx)}
			if diagnostics := state.Set(ctx, &test.state); diagnostics.HasError() {
				t.Fatalf("set state: %v", diagnostics)
			}
			plan := tfsdk.Plan{Schema: workers_build_token.ResourceSchema(ctx)}
			if diagnostics := plan.Set(ctx, &test.plan); diagnostics.HasError() {
				t.Fatalf("set plan: %v", diagnostics)
			}
			response := &resource.UpdateResponse{State: state}
			newBuildTokenResource(t, server.URL).Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, response)
			if !response.Diagnostics.HasError() {
				t.Fatal("unexpected update must be rejected")
			}
		})
	}
}

func TestBuildTokenDeleteMapsRequestAndTreats404AsSuccess(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodDelete {
					t.Errorf("method = %s, want DELETE", request.Method)
				}
				wantPath := "/accounts/" + tokenTestAccountID + "/builds/tokens/" + tokenTestUUID
				if request.URL.Path != wantPath {
					t.Errorf("path = %s, want %s", request.URL.Path, wantPath)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				fmt.Fprint(w, `{"success":true,"result":null}`)
			}))
			defer server.Close()

			state := tokenState(t, ctx, types.StringValue(tokenTestSecret))
			response := &resource.DeleteResponse{State: state}
			newBuildTokenResource(t, server.URL).Delete(ctx, resource.DeleteRequest{State: state}, response)
			if response.Diagnostics.HasError() {
				t.Fatalf("delete diagnostics: %v", response.Diagnostics)
			}
		})
	}
}

func TestBuildTokenDeleteRejectsUnsuccessfulEnvelopeWithoutLeakingDetails(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":false,"errors":[{"code":12000,"message":"echo %s"}],"result":null}`, tokenTestSecret)
	}))
	defer server.Close()

	state := tokenState(t, ctx, types.StringValue(tokenTestSecret))
	response := &resource.DeleteResponse{State: state}
	newBuildTokenResource(t, server.URL).Delete(ctx, resource.DeleteRequest{State: state}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("delete must reject a success=false envelope")
	}
	if strings.Contains(diagnosticText(response.Diagnostics), tokenTestSecret) {
		t.Fatal("delete diagnostics leaked a sensitive response detail")
	}
}
