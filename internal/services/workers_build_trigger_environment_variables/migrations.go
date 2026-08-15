package workers_build_trigger_environment_variables

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.ResourceWithUpgradeState = (*WorkersBuildTriggerEnvironmentVariablesResource)(nil)

func (r *WorkersBuildTriggerEnvironmentVariablesResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	targetSchema := ResourceSchema(ctx)
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &targetSchema,
			StateUpgrader: func(_ context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				resp.State.Raw = req.State.Raw
			},
		},
	}
}
