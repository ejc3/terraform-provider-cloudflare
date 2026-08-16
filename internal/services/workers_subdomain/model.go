package workers_subdomain

import "github.com/hashicorp/terraform-plugin-framework/types"

// WorkersSubdomainModel represents the account-wide workers.dev label. It is
// deliberately separate from workers_script_subdomain, which controls whether
// an individual script is exposed on that label.
type WorkersSubdomainModel struct {
	ID        types.String `tfsdk:"id"`
	AccountID types.String `tfsdk:"account_id"`
	Subdomain types.String `tfsdk:"subdomain"`
}
