package workers_subdomain

import "github.com/hashicorp/terraform-plugin-framework/types"

type WorkersSubdomainDataSourceModel struct {
	AccountID types.String `tfsdk:"account_id"`
	Subdomain types.String `tfsdk:"subdomain"`
}
