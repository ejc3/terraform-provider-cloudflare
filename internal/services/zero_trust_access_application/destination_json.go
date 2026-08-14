package zero_trust_access_application

import (
	"fmt"
	"strings"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/apijson"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MarshalJSONWithState serializes a destination as the API's discriminated
// union. Keeping the variants explicit prevents fields from a previously
// configured destination type from leaking into an update request.
func (m ZeroTrustAccessApplicationDestinationsModel) MarshalJSONWithState(_ any, _ any) ([]byte, error) {
	destinationType := strings.ToLower(m.Type.ValueString())
	typeValue := m.Type
	if !m.Type.IsNull() && !m.Type.IsUnknown() {
		typeValue = types.StringValue(destinationType)
	}

	switch destinationType {
	case "", "public":
		return apijson.Marshal(struct {
			Type types.String `json:"type,optional"`
			URI  types.String `json:"uri,optional"`
		}{
			Type: typeValue,
			URI:  m.URI,
		})
	case "private":
		return apijson.Marshal(struct {
			Type       types.String `json:"type,required"`
			CIDR       types.String `json:"cidr,optional"`
			Hostname   types.String `json:"hostname,optional"`
			L4Protocol types.String `json:"l4_protocol,optional"`
			PortRange  types.String `json:"port_range,optional"`
			VnetID     types.String `json:"vnet_id,optional"`
		}{
			Type:       typeValue,
			CIDR:       m.CIDR,
			Hostname:   m.Hostname,
			L4Protocol: m.L4Protocol,
			PortRange:  m.PortRange,
			VnetID:     m.VnetID,
		})
	case "via_mcp_server_portal":
		return apijson.Marshal(struct {
			Type        types.String `json:"type,required"`
			McpServerID types.String `json:"mcp_server_id,required"`
		}{
			Type:        typeValue,
			McpServerID: m.McpServerID,
		})
	case "worker", "preview_worker":
		return apijson.Marshal(struct {
			Type     types.String `json:"type,required"`
			WorkerID types.String `json:"worker_id,required"`
		}{
			Type:     typeValue,
			WorkerID: m.WorkerID,
		})
	case "all_workers", "all_preview_workers":
		return apijson.Marshal(struct {
			Type types.String `json:"type,required"`
		}{
			Type: typeValue,
		})
	default:
		return nil, fmt.Errorf("unsupported Access destination type %q", m.Type.ValueString())
	}
}
