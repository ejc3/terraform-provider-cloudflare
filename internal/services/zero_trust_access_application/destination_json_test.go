package zero_trust_access_application_test

import (
	"context"
	"os"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/apijson"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/zero_trust_access_application"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZeroTrustAccessApplicationDestinationMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"public":                `{"type":"public","uri":"example.com/admin"}`,
		"private":               `{"cidr":"10.0.0.0/24","hostname":"origin.example.com","l4_protocol":"tcp","port_range":"443","type":"private","vnet_id":"vnet-id"}`,
		"via_mcp_server_portal": `{"mcp_server_id":"mcp-server-id","type":"via_mcp_server_portal"}`,
		"worker":                `{"type":"worker","worker_id":"worker-id"}`,
		"preview_worker":        `{"type":"preview_worker","worker_id":"worker-id"}`,
		"all_workers":           `{"type":"all_workers"}`,
		"all_preview_workers":   `{"type":"all_preview_workers"}`,
	}

	for destinationType, expected := range tests {
		destinationType := destinationType
		expected := expected
		t.Run(destinationType, func(t *testing.T) {
			t.Parallel()

			// Populate every union field to prove the serializer emits only the
			// fields valid for the selected destination variant.
			model := zero_trust_access_application.ZeroTrustAccessApplicationDestinationsModel{
				Type:        types.StringValue(destinationType),
				URI:         types.StringValue("example.com/admin"),
				CIDR:        types.StringValue("10.0.0.0/24"),
				Hostname:    types.StringValue("origin.example.com"),
				L4Protocol:  types.StringValue("tcp"),
				PortRange:   types.StringValue("443"),
				VnetID:      types.StringValue("vnet-id"),
				McpServerID: types.StringValue("mcp-server-id"),
				WorkerID:    types.StringValue("worker-id"),
			}

			actual, err := apijson.Marshal(model)
			require.NoError(t, err)
			assert.JSONEq(t, expected, string(actual))
		})
	}
}

func TestZeroTrustAccessApplicationWorkerDestinationUnmarshalJSON(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/access_application_worker_response.json")
	require.NoError(t, err)

	var envelope zero_trust_access_application.ZeroTrustAccessApplicationResultEnvelope
	require.NoError(t, apijson.Unmarshal(fixture, &envelope))

	destinations, diags := envelope.Result.Destinations.AsStructSliceT(context.Background())
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	require.Len(t, destinations, 1)
	assert.Equal(t, "worker", destinations[0].Type.ValueString())
	assert.Equal(t, "72edf31f83e240448fce38bef56104e3", destinations[0].WorkerID.ValueString())
	assert.True(t, envelope.Result.Domain.IsNull(), "worker destination response must not synthesize a domain")
}
