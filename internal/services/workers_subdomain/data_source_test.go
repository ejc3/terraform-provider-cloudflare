package workers_subdomain_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/workers_subdomain"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkersSubdomainDataSourceRead(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assertRequest(t, req, http.MethodGet)
		writeJSON(t, w, http.StatusOK, workersSubdomainEnvelope("cc-games"))
	}))
	defer ts.Close()

	d := workers_subdomain.NewWorkersSubdomainDataSource().(*workers_subdomain.WorkersSubdomainDataSource)
	client := cloudflare.NewClient(
		option.WithBaseURL(ts.URL),
		option.WithAPIToken(testToken),
	)
	configureResp := &datasource.ConfigureResponse{}
	d.Configure(ctx, datasource.ConfigureRequest{ProviderData: client}, configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("configure data source: %v", configureResp.Diagnostics)
	}

	schema := workers_subdomain.DataSourceSchema(ctx)
	configState := tfsdk.State{Schema: schema}
	configModel := workers_subdomain.WorkersSubdomainDataSourceModel{
		AccountID: types.StringValue(testAccountID),
		Subdomain: types.StringNull(),
	}
	if diags := configState.Set(ctx, &configModel); diags.HasError() {
		t.Fatalf("set data source config: %v", diags)
	}
	config := tfsdk.Config{Schema: schema, Raw: configState.Raw}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schema}}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var got workers_subdomain.WorkersSubdomainDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get data source state: %v", diags)
	}
	if got.AccountID.ValueString() != testAccountID {
		t.Errorf("account_id = %q, want %q", got.AccountID.ValueString(), testAccountID)
	}
	if got.Subdomain.ValueString() != "cc-games" {
		t.Errorf("subdomain = %q, want cc-games", got.Subdomain.ValueString())
	}
}
