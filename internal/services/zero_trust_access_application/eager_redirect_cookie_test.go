package zero_trust_access_application_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/acctest"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/apijson"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/services/zero_trust_access_application"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The setting reaches the API only when configured: an unset value leaves Cloudflare's
// default (true) in place instead of sending false.
func TestZeroTrustAccessApplicationEagerRedirectCookieMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value    types.Bool
		expected string
	}{
		"true":  {types.BoolValue(true), `{"name":"app","type":"self_hosted","eager_redirect_cookie_setting":true}`},
		"false": {types.BoolValue(false), `{"name":"app","type":"self_hosted","eager_redirect_cookie_setting":false}`},
		"unset": {types.BoolNull(), `{"name":"app","type":"self_hosted"}`},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := zero_trust_access_application.ZeroTrustAccessApplicationModel{
				Name:                       types.StringValue("app"),
				Type:                       types.StringValue("self_hosted"),
				EagerRedirectCookieSetting: test.value,
			}

			actual, err := model.MarshalJSON()
			require.NoError(t, err)
			assert.JSONEq(t, test.expected, string(actual))
		})
	}
}

// The API's value is read back into state, so a change made outside Terraform shows in a plan.
func TestZeroTrustAccessApplicationEagerRedirectCookieUnmarshalJSON(t *testing.T) {
	t.Parallel()

	for _, value := range []bool{true, false} {
		value := value
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			t.Parallel()

			var model zero_trust_access_application.ZeroTrustAccessApplicationModel
			body := fmt.Sprintf(`{"id":"app-id","name":"app","type":"self_hosted","eager_redirect_cookie_setting":%t}`, value)
			require.NoError(t, apijson.Unmarshal([]byte(body), &model))
			assert.Equal(t, types.BoolValue(value), model.EagerRedirectCookieSetting)
		})
	}
}

func TestZeroTrustAccessApplicationEagerRedirectCookieValidation(t *testing.T) {
	const accountID = "12ea67fb7ced068de03f35c22688e436"

	tests := map[string]struct {
		config      string
		expectError *regexp.Regexp
	}{
		"self-hosted application with two hostnames": {
			config: eagerRedirectCookieConfig(accountID, "self_hosted", "true"),
		},
		"self-hosted application turning it off": {
			config: eagerRedirectCookieConfig(accountID, "self_hosted", "false"),
		},
		"rejected on a SaaS application": {
			config:      eagerRedirectCookieConfig(accountID, "saas", "true"),
			expectError: regexp.MustCompile(`(?s)eager_redirect_cookie_setting.*can only be set if.*type.*self_hosted`),
		},
		"rejected on a bookmark": {
			config:      eagerRedirectCookieConfig(accountID, "bookmark", "true"),
			expectError: regexp.MustCompile(`(?s)eager_redirect_cookie_setting.*can only be set if.*type.*self_hosted`),
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

func eagerRedirectCookieConfig(accountID, applicationType, value string) string {
	return fmt.Sprintf(`
resource "cloudflare_zero_trust_access_application" "test" {
  account_id                    = %q
  name                          = "Two hostnames, one session"
  type                          = %q
  eager_redirect_cookie_setting = %s

  destinations = [
    { type = "public", uri = "app.example.com" },
    { type = "public", uri = "api.example.com" },
  ]
}
`, accountID, applicationType, value)
}
