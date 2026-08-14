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
		"all workers": {
			config: workerDestinationConfig(accountID, "all_workers", false),
		},
		"all preview workers": {
			config: workerDestinationConfig(accountID, "all_preview_workers", false),
		},
		"worker missing worker id": {
			config:      workerDestinationConfig(accountID, "worker", false),
			expectError: regexp.MustCompile(`worker_id.*has to be set.*worker.*preview_worker`),
		},
		"all workers rejects worker id": {
			config:      workerDestinationConfig(accountID, "all_workers", true),
			expectError: regexp.MustCompile(`worker_id.*can only be set.*worker.*preview_worker`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      test.config,
					PlanOnly:    true,
					ExpectError: test.expectError,
				}},
			})
		})
	}
}

func workerDestinationConfig(accountID, destinationType string, withWorkerID bool) string {
	workerID := ""
	if withWorkerID {
		workerID = `worker_id = "72edf31f83e240448fce38bef56104e3"`
	}

	return fmt.Sprintf(`
resource "cloudflare_zero_trust_access_application" "test" {
  account_id = %q
  name       = "Worker-native Access"
  type       = "self_hosted"

  destinations = [{
    type = %q
    %s
  }]
}
`, accountID, destinationType, workerID)
}
