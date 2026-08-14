package zero_trust_access_application_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestZeroTrustAccessApplicationWorkerDestinationValidation(t *testing.T) {
	const accountID = "12ea67fb7ced068de03f35c22688e436"

	tests := map[string]struct {
		config      string
		expectError *regexp.Regexp
	}{
		"worker without domain": {
			config: acctest.LoadTestCase("accessapplicationconfigwithworkerdestination.tf", "test", accountID, "72edf31f83e240448fce38bef56104e3"),
		},
		"preview worker": {
			config: workerDestinationConfig(accountID, "preview_worker", true),
		},
		"mixed case worker is rejected": {
			config:      workerDestinationConfig(accountID, "WORKER", true),
			expectError: regexp.MustCompile(`(?s)Invalid Worker Destination Type.*canonical lowercase spelling.*worker`),
		},
		"all workers": {
			config: workerDestinationConfig(accountID, "all_workers", false),
		},
		"all preview workers": {
			config: workerDestinationConfig(accountID, "all_preview_workers", false),
		},
		"worker missing worker id": {
			config:      workerDestinationConfig(accountID, "worker", false),
			expectError: regexp.MustCompile(`(?s)worker_id.*has to be set.*worker.*preview_worker`),
		},
		"mixed case worker missing worker id": {
			config:      workerDestinationConfig(accountID, "WoRkEr", false),
			expectError: regexp.MustCompile(`(?s)worker_id.*has to be set.*worker.*preview_worker`),
		},
		"all workers rejects worker id": {
			config:      workerDestinationConfig(accountID, "all_workers", true),
			expectError: regexp.MustCompile(`(?s)worker_id.*can only be set.*worker.*preview_worker`),
		},
		"worker rejects ssh application": {
			config:      workerDestinationConfigForApplication(accountID, "ssh", "worker", true),
			expectError: regexp.MustCompile(`(?s)destinations\[0\]\.type.*Worker destinations.*self-hosted`),
		},
		"preview worker rejects rdp application": {
			config:      workerDestinationConfigForApplication(accountID, "rdp", "preview_worker", true),
			expectError: regexp.MustCompile(`(?s)destinations\[0\]\.type.*Worker destinations.*self-hosted`),
		},
		"all workers rejects mcp portal application": {
			config:      workerDestinationConfigForApplication(accountID, "mcp_portal", "all_workers", false),
			expectError: regexp.MustCompile(`(?s)destinations\[0\]\.type.*Worker destinations.*self-hosted`),
		},
		"all preview workers rejects mcp application": {
			config:      workerDestinationConfigForApplication(accountID, "mcp", "all_preview_workers", false),
			expectError: regexp.MustCompile(`(?s)destinations\[0\]\.type.*Worker destinations.*self-hosted`),
		},
		"public destination remains valid for ssh": {
			config: workerDestinationConfigForApplication(accountID, "ssh", "public", false),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:             test.config,
					PlanOnly:           true,
					ExpectNonEmptyPlan: test.expectError == nil,
					ExpectError:        test.expectError,
				}},
			})
		})
	}
}

func workerDestinationConfig(accountID, destinationType string, withWorkerID bool) string {
	return workerDestinationConfigForApplication(accountID, "self_hosted", destinationType, withWorkerID)
}

func workerDestinationConfigForApplication(accountID, applicationType, destinationType string, withWorkerID bool) string {
	workerID := ""
	if withWorkerID {
		workerID = `worker_id = "72edf31f83e240448fce38bef56104e3"`
	}

	return fmt.Sprintf(`
resource "cloudflare_zero_trust_access_application" "test" {
  account_id = %q
  name       = "Worker-native Access"
  type       = %q

  destinations = [{
    type = %q
    %s
  }]
}
`, accountID, applicationType, destinationType, workerID)
}
