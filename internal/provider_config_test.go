package internal

import (
	"os"
	"testing"

	"github.com/cloudflare/terraform-provider-cloudflare/internal/consts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConfiguredOperatorSuffix(t *testing.T) {
	t.Setenv(consts.UserAgentOperatorSuffixEnvVarKey, "environment/operator")

	tests := []struct {
		name  string
		value types.String
		want  string
	}{
		{name: "configuration takes precedence", value: types.StringValue("configuration/operator"), want: "configuration/operator"},
		{name: "null uses environment", value: types.StringNull(), want: "environment/operator"},
		{name: "unknown uses environment", value: types.StringUnknown(), want: "environment/operator"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := configuredOperatorSuffix(test.value)
			if !ok || got != test.want {
				t.Fatalf("configuredOperatorSuffix() = %q, %t; want %q, true", got, ok, test.want)
			}
		})
	}
}

func TestConfiguredOperatorSuffixAbsent(t *testing.T) {
	previous, existed := os.LookupEnv(consts.UserAgentOperatorSuffixEnvVarKey)
	if err := os.Unsetenv(consts.UserAgentOperatorSuffixEnvVarKey); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(consts.UserAgentOperatorSuffixEnvVarKey, previous)
		} else {
			_ = os.Unsetenv(consts.UserAgentOperatorSuffixEnvVarKey)
		}
	})

	if got, ok := configuredOperatorSuffix(types.StringNull()); ok {
		t.Fatalf("configuredOperatorSuffix() = %q, true; want absent", got)
	}
}
