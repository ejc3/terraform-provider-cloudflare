// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package moq_relay

import (
	"github.com/cloudflare/terraform-provider-cloudflare/internal/apijson"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/customfield"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type MoQRelayResultEnvelope struct {
	Result MoQRelayModel `json:"result"`
}

type MoQRelayModel struct {
	ID        types.String                                       `tfsdk:"id" json:"-,computed"`
	UID       types.String                                       `tfsdk:"uid" json:"uid,computed"`
	AccountID types.String                                       `tfsdk:"account_id" path:"account_id,required"`
	Name      types.String                                       `tfsdk:"name" json:"name,required"`
	Config    customfield.NestedObject[MoQRelayConfigModel]      `tfsdk:"config" json:"config,computed_optional"`
	Created   timetypes.RFC3339                                  `tfsdk:"created" json:"created,computed" format:"date-time"`
	Modified  timetypes.RFC3339                                  `tfsdk:"modified" json:"modified,computed" format:"date-time"`
	Status    types.String                                       `tfsdk:"status" json:"status,computed"`
	Issuers   customfield.NestedObjectList[MoQRelayIssuersModel] `tfsdk:"issuers" json:"issuers,computed,no_refresh"`
}

func (m MoQRelayModel) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(m)
}

func (m MoQRelayModel) MarshalJSONForUpdate(state MoQRelayModel) (data []byte, err error) {
	return apijson.MarshalForUpdate(m, state)
}

type MoQRelayConfigModel struct {
	Upstreams customfield.NestedObject[MoQRelayConfigUpstreamsModel] `tfsdk:"upstreams" json:"upstreams,computed_optional"`
}

type MoQRelayConfigUpstreamsModel struct {
	Enabled   types.Bool                                                          `tfsdk:"enabled" json:"enabled,computed_optional"`
	Upstreams customfield.NestedObjectList[MoQRelayConfigUpstreamsUpstreamsModel] `tfsdk:"upstreams" json:"upstreams,computed_optional"`
}

type MoQRelayConfigUpstreamsUpstreamsModel struct {
	URL types.String `tfsdk:"url" json:"url,optional"`
}

type MoQRelayIssuersModel struct {
	CloudflareTokens customfield.NestedObjectList[MoQRelayIssuersCloudflareTokensModel] `tfsdk:"cloudflare_tokens" json:"cloudflare_tokens,computed"`
	Issuer           types.String                                                       `tfsdk:"issuer" json:"issuer,computed"`
	Type             types.String                                                       `tfsdk:"type" json:"type,computed"`
}

type MoQRelayIssuersCloudflareTokensModel struct {
	Created    timetypes.RFC3339              `tfsdk:"created" json:"created,computed" format:"date-time"`
	Expires    timetypes.RFC3339              `tfsdk:"expires" json:"expires,computed" format:"date-time"`
	Jti        types.String                   `tfsdk:"jti" json:"jti,computed"`
	Operations customfield.List[types.String] `tfsdk:"operations" json:"operations,computed"`
	Label      types.String                   `tfsdk:"label" json:"label,computed"`
	Secret     types.String                   `tfsdk:"secret" json:"secret,computed"`
}
