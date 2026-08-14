package zero_trust_access_application

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type canonicalWorkerDestinationTypeValidator struct{}

var _ validator.String = canonicalWorkerDestinationTypeValidator{}

func (canonicalWorkerDestinationTypeValidator) Description(context.Context) string {
	return "Worker destination types must use their canonical lowercase spelling."
}

func (v canonicalWorkerDestinationTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v canonicalWorkerDestinationTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, res *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	destinationType := req.ConfigValue.ValueString()
	canonicalType := strings.ToLower(destinationType)
	if isWorkerDestinationType(destinationType) && destinationType != canonicalType {
		res.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Worker Destination Type",
			fmt.Sprintf("Worker destination type %q must use its canonical lowercase spelling %q.", destinationType, canonicalType),
		)
	}
}

type workerDestinationApplicationTypeValidator struct{}

var _ validator.String = workerDestinationApplicationTypeValidator{}

func (workerDestinationApplicationTypeValidator) Description(context.Context) string {
	return "Worker destinations can only be used with self-hosted Access applications."
}

func (v workerDestinationApplicationTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v workerDestinationApplicationTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, res *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || !isWorkerDestinationType(req.ConfigValue.ValueString()) {
		return
	}

	var applicationType types.String
	res.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &applicationType)...)
	if res.Diagnostics.HasError() || applicationType.IsNull() || applicationType.IsUnknown() {
		return
	}

	if !strings.EqualFold(applicationType.ValueString(), "self_hosted") {
		description := fmt.Sprintf("%q %s", req.Path, v.Description(ctx))
		res.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(req.Path, description))
	}
}

func isWorkerDestinationType(destinationType string) bool {
	switch strings.ToLower(destinationType) {
	case "worker", "preview_worker", "all_workers", "all_preview_workers":
		return true
	default:
		return false
	}
}
