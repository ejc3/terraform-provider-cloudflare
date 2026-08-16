# EJC Cloudflare Terraform Provider Fork

The [`ejc3/cloudflare` community provider](https://registry.terraform.io/providers/ejc3/cloudflare/latest/docs) is a focused fork of the
[official Cloudflare Terraform provider](https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs). It provides convenient
access to the [Cloudflare REST API](https://developers.cloudflare.com/api) and carries the Worker-native Access and Workers Builds support
needed by the EJC development fleet while those capabilities are not available upstream.

## Requirements

This provider requires Terraform CLI 1.0 or later. You can [install it for your system](https://developer.hashicorp.com/terraform/install)
on Hashicorp's website.

## Usage

Add the following to your `main.tf` file:

```hcl
# Declare the provider and version
terraform {
  required_providers {
    cloudflare = {
      source  = "ejc3/cloudflare"
      version = "= 5.24.0"
    }
  }
}

# Initialize the provider
provider "cloudflare" {
  # The preferred authorization scheme for interacting with the Cloudflare API. [Create a token](https://developers.cloudflare.com/fundamentals/api/get-started/create-token/).
  api_token = "Sn3lZJTBX6kkg7OdcBUAxOO963GEIyGQqnFTOFYY" # or set CLOUDFLARE_API_TOKEN env variable
}

# Configure a resource
resource "cloudflare_zone" "example_zone" {
  account = {
    id = "023e105f4ecef8ad9ca31a8372d0c353"
  }
  name = "example.com"
  type = "full"
}
```

Initialize your project by running `terraform init` in the directory.

Additional examples can be found in the [./examples](./examples) folder within this repository, and you can
refer to the full documentation on [the Terraform Registry](https://registry.terraform.io/providers/ejc3/cloudflare/latest/docs).

### Provider Options

When you initialize the provider, the following options are supported. It is recommended to use environment variables for sensitive values like access tokens.
If an environment variable is provided, then the option does not need to be set in the terraform source.

| Property                     | Environment variable                       | Required | Default value |
| ---------------------------- | ------------------------------------------ | -------- | ------------- |
| api_token                    | `CLOUDFLARE_API_TOKEN`                     | false    | —             |
| api_key                      | `CLOUDFLARE_API_KEY`                       | false    | —             |
| email                        | `CLOUDFLARE_EMAIL`                         | false    | —             |
| api_user_service_key         | `CLOUDFLARE_API_USER_SERVICE_KEY`          | false    | —             |
| base_url                     | `CLOUDFLARE_BASE_URL`                      | false    | —             |
| user_agent_operator_suffix   | `CLOUDFLARE_USER_AGENT_OPERATOR_SUFFIX`    | false    | —             |

## Semantic versioning

This package generally follows [SemVer](https://semver.org/spec/v2.0.0.html) conventions, though certain backwards-incompatible changes may be released as minor versions:

1. Changes to library internals which are technically public but not intended or documented for external use. _(Please open a GitHub issue to let us know if you are relying on such internals.)_
2. Changes that we do not expect to impact the vast majority of users in practice.

We take backwards-compatibility seriously and work hard to ensure you can rely on a smooth upgrade experience.

For fork-specific questions, bugs, or suggestions, please open an [issue](https://github.com/ejc3/terraform-provider-cloudflare/issues). Upstream-provider issues remain in the [Cloudflare repository](https://github.com/cloudflare/terraform-provider-cloudflare/issues).

## Maintenance

This SDK is actively maintained, however, many issues are tracked outside of GitHub on internal Cloudflare systems. Members of the community are welcome to join and discuss your issues during our weekly triage meetings. For urgent issues, please contact [Cloudflare support](https://developers.cloudflare.com/support/contacting-cloudflare-support/). 

* [Community triage meeting](https://calendar.google.com/calendar/embed?src=c_dbf6ce250643f2e60f806d28f3fc09a9de24cbe0ab3ffb699838303d2adfc9e4%40group.calendar.google.com&ctz=America%2FLos_Angeles)

## Contributing

See [the contributing documentation](./CONTRIBUTING.md).
