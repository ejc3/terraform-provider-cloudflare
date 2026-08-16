package workers_build_repository_connection_test

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
	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_build_repository_connection"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	repositoryTestAccountID = "12ea67fb7ced068de03f35c22688e436"
	repositoryTestUUID      = "182bd5e5-6e1a-4fe4-a799-aa6d9a6ab26e"
)

func newRepositoryConnectionResource(t *testing.T, baseURL string) *workers_build_repository_connection.WorkersBuildRepositoryConnectionResource {
	t.Helper()
	instance := workers_build_repository_connection.NewResource().(*workers_build_repository_connection.WorkersBuildRepositoryConnectionResource)
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

func repositoryConnectionModel(id string) workers_build_repository_connection.WorkersBuildRepositoryConnectionModel {
	idValue := types.StringUnknown()
	if id != "" {
		idValue = types.StringValue(id)
	}
	return workers_build_repository_connection.WorkersBuildRepositoryConnectionModel{
		ID:                  idValue,
		AccountID:           types.StringValue(repositoryTestAccountID),
		ProviderType:        types.StringValue("github"),
		ProviderAccountID:   types.StringValue("250920182"),
		ProviderAccountName: types.StringValue("CoderColton"),
		RepoID:              types.StringValue("1120877379"),
		RepoName:            types.StringValue("colton-games"),
	}
}

func repositoryConnectionPlan(t *testing.T, ctx context.Context) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: workers_build_repository_connection.ResourceSchema(ctx)}
	if diagnostics := plan.Set(ctx, repositoryConnectionModel("")); diagnostics.HasError() {
		t.Fatalf("set plan: %v", diagnostics)
	}
	return plan
}

func repositoryConnectionState(t *testing.T, ctx context.Context) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: workers_build_repository_connection.ResourceSchema(ctx)}
	if diagnostics := state.Set(ctx, repositoryConnectionModel(repositoryTestUUID)); diagnostics.HasError() {
		t.Fatalf("set state: %v", diagnostics)
	}
	return state
}

func TestRepositoryConnectionCreateMapsRequestAndResponse(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", request.Method)
		}
		if request.URL.Path != "/accounts/"+repositoryTestAccountID+"/builds/repos/connections" {
			t.Errorf("path = %s", request.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		want := map[string]string{
			"provider_type":         "github",
			"provider_account_id":   "250920182",
			"provider_account_name": "CoderColton",
			"repo_id":               "1120877379",
			"repo_name":             "colton-games",
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("%s = %q, want %q", key, body[key], value)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"result":{"repo_connection_uuid":%q,"provider_type":"github","provider_account_id":"250920182","provider_account_name":"CoderColton","repo_id":"1120877379","repo_name":"colton-games"}}`, repositoryTestUUID)
	}))
	defer server.Close()

	instance := newRepositoryConnectionResource(t, server.URL)
	response := &resource.CreateResponse{State: tfsdk.State{Schema: workers_build_repository_connection.ResourceSchema(ctx)}}
	instance.Create(ctx, resource.CreateRequest{Plan: repositoryConnectionPlan(t, ctx)}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", response.Diagnostics)
	}

	var result workers_build_repository_connection.WorkersBuildRepositoryConnectionModel
	response.Diagnostics.Append(response.State.Get(ctx, &result)...)
	if result.ID.ValueString() != repositoryTestUUID {
		t.Errorf("id = %q, want %q", result.ID.ValueString(), repositoryTestUUID)
	}
}

func TestRepositoryConnectionCreateFailureExplainsGitHubAppAuthorization(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success":false,"errors":[{"code":12000,"message":"repository installation not found"}]}`)
	}))
	defer server.Close()

	response := &resource.CreateResponse{State: tfsdk.State{Schema: workers_build_repository_connection.ResourceSchema(ctx)}}
	newRepositoryConnectionResource(t, server.URL).Create(ctx, resource.CreateRequest{Plan: repositoryConnectionPlan(t, ctx)}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("expected create error")
	}
	var details strings.Builder
	for _, diagnostic := range response.Diagnostics.Errors() {
		details.WriteString(diagnostic.Detail())
	}
	message := details.String()
	if !strings.Contains(message, "Cloudflare Workers and Pages GitHub App") || !strings.Contains(message, "CoderColton/colton-games") {
		t.Fatalf("create error is not actionable: %s", message)
	}
}

func TestRepositoryConnectionReadPreservesStateWithoutRequest(t *testing.T) {
	ctx := context.Background()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	state := repositoryConnectionState(t, ctx)
	response := &resource.ReadResponse{State: state}
	newRepositoryConnectionResource(t, server.URL).Read(ctx, resource.ReadRequest{State: state}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", response.Diagnostics)
	}
	if requests != 0 {
		t.Fatalf("read made %d API requests; Cloudflare has no read endpoint", requests)
	}

	var result workers_build_repository_connection.WorkersBuildRepositoryConnectionModel
	response.Diagnostics.Append(response.State.Get(ctx, &result)...)
	if result.ID.ValueString() != repositoryTestUUID || result.RepoName.ValueString() != "colton-games" {
		t.Fatalf("read did not preserve state: %#v", result)
	}
}

func TestRepositoryConnectionDeleteMapsRequestAndTreats404AsSuccess(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodDelete {
					t.Errorf("method = %s, want DELETE", request.Method)
				}
				wantPath := "/accounts/" + repositoryTestAccountID + "/builds/repos/connections/" + repositoryTestUUID
				if request.URL.Path != wantPath {
					t.Errorf("path = %s, want %s", request.URL.Path, wantPath)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				fmt.Fprint(w, `{"success":true,"result":null}`)
			}))
			defer server.Close()

			state := repositoryConnectionState(t, ctx)
			response := &resource.DeleteResponse{State: state}
			newRepositoryConnectionResource(t, server.URL).Delete(ctx, resource.DeleteRequest{State: state}, response)
			if response.Diagnostics.HasError() {
				t.Fatalf("delete diagnostics: %v", response.Diagnostics)
			}
		})
	}
}

func TestRepositoryConnectionDeleteRejectsUnsuccessfulEnvelope(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":false,"errors":[{"code":12000,"message":"not deleted"}],"result":null}`)
	}))
	defer server.Close()

	state := repositoryConnectionState(t, ctx)
	response := &resource.DeleteResponse{State: state}
	newRepositoryConnectionResource(t, server.URL).Delete(ctx, resource.DeleteRequest{State: state}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("delete must reject a success=false envelope")
	}
}

func TestRepositoryConnectionImportIsExplicitlyRejected(t *testing.T) {
	response := &resource.ImportStateResponse{}
	workers_build_repository_connection.NewResource().(*workers_build_repository_connection.WorkersBuildRepositoryConnectionResource).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: repositoryTestAccountID + "/" + repositoryTestUUID},
		response,
	)
	if !response.Diagnostics.HasError() {
		t.Fatal("import must be rejected because Cloudflare has no read endpoint")
	}
}
