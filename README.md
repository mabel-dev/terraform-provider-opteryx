# terraform-provider-opteryx

[![build](https://github.com/mabel-dev/terraform-provider-opteryx/actions/workflows/build.yml/badge.svg)](https://github.com/mabel-dev/terraform-provider-opteryx/actions/workflows/build.yml)

A Terraform provider for managing [policy.opteryx](https://github.com/mabel-dev/policy.opteryx.app) access policies as code, instead of through ad-hoc API calls.

## Status

- `opteryx_access_policy` (resource) — a single principal/role/pattern grant, backed by `policy.opteryx`'s `/v1/access/workspace/{workspace}/policies...` CRUD endpoints.
- `opteryx_access_policies` (data source) — lists every policy in a workspace, backed by the list endpoint. Useful for auditing existing grants or finding a policy's `id` to import.

Not published anywhere — used locally via Terraform's `dev_overrides` (see Usage below). No public registry publishing is planned; that would need a HashiCorp registry account but not any other involvement from HashiCorp.

## Requirements

- [Go](https://go.dev/) matching the `go` directive in `go.mod` (currently 1.25+)
- [Terraform](https://developer.hashicorp.com/terraform) 1.0+ (for local testing with the CLI)

## Building

```bash
make build    # go build -o terraform-provider-opteryx
make install  # builds, then go install's it onto $GOPATH/bin
```

No Go installed locally? Push a branch or open a PR — [`.github/workflows/build.yml`](.github/workflows/build.yml) runs `go mod tidy`, `gofmt`, `go vet`, `go build`, and `go test` on every push and pull request against `main`.

## Usage

This isn't published to any registry, so Terraform can't find it via a normal `terraform init`. Point Terraform at your locally built binary instead, using [`dev_overrides`](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers):

1. Build and install it:

   ```bash
   make install   # go build, then go install — lands in $(go env GOPATH)/bin
   ```

2. Add a dev override to `~/.terraformrc` (create the file if it doesn't exist):

   ```hcl
   provider_installation {
     dev_overrides {
       "mabel-dev/opteryx" = "/Users/you/go/bin"  # $(go env GOPATH)/bin
     }
     direct {}
   }
   ```

3. Write config that uses it — see [`examples/`](examples/) for a starting `provider.tf` and `opteryx_access_policy` resources. With `dev_overrides` active you skip `terraform init` for this provider entirely (Terraform prints a warning that overrides are in effect and skips its lock-file check).

4. Get a token. `token` must be a bearer JWT for an owner/admin identity that is **not** a principal in any policy this config manages. `authenticate.opteryx`'s access tokens are short-lived — 5 minutes (`ACCESS_TOKEN_EXPIRATION_MINUTES` in its `app/core.py`) — so exporting one by hand will likely expire mid-`apply`. Mint one immediately before each run instead, e.g. with a client-credentials identity (see `authenticate.opteryx`'s `POST /clients/{client_id}/credentials` to create one, `POST /token` to exchange it):

   ```bash
   ./examples/run-terraform.sh plan
   ./examples/run-terraform.sh apply
   ```

   That wrapper fetches a fresh `OPTERYX_TOKEN` from `OPTERYX_CLIENT_ID`/`OPTERYX_CLIENT_SECRET` and execs `terraform "$@"`.

## Configuration

```hcl
provider "opteryx" {
  endpoint = "https://policy.opteryx.app" # optional, or OPTERYX_ENDPOINT
  token    = var.opteryx_token            # required, or OPTERYX_TOKEN
}
```

`token` is a bearer JWT minted by `authenticate.opteryx` (the same kind clients present to any opteryx service). Two things to know before pointing this at a real workspace:

1. **Self-grant/self-modify is rejected at the API, not just in the UI.** `policy.opteryx` refuses to let an identity create, update, or delete a policy where it is the principal (`app/routes/v1/access.py`, the `create_policy`/`update_policy`/`delete_policy` handlers). If the identity behind your token ever appears as a `principal` in the same config, `apply` will fail with a 403 from the service, not a Terraform-side validation error. Use a distinct owner/admin identity to run Terraform.
2. **`role` is validated here but not everywhere downstream.** `authenticate.opteryx` mints whatever string is stored into the JWT's `policies` claim without checking it against the `owner/admin/writer/reader` enum (see the fleet's [Opteryx services map](../policy.opteryx) notes) — `policy.opteryx`'s own validation, which this provider's schema mirrors via `stringvalidator.OneOf`, is the only thing stopping a bad role from reaching production tokens. Don't bypass it by hand-editing state.

## Resource: `opteryx_access_policy`

```hcl
resource "opteryx_access_policy" "analytics_reader" {
  workspace = "analytics"
  principal = "jane@example.com"
  role      = "reader"
  pattern   = "analytics.sales.*"
}
```

| Attribute    | Type   | Notes |
|--------------|--------|-------|
| `workspace`  | string | Required. Forces replacement — the API has no move/rename. |
| `principal`  | string | Required. Forces replacement — `PUT` has no way to reassign a policy's principal. `"*"` means any authenticated user. |
| `role`       | string | Required. One of `owner`, `admin`, `writer`, `reader`. Updatable in place. |
| `pattern`    | string | Required. Updatable in place. A wildcard `principal` ("*") must pair with an exact, non-glob pattern. |
| `id`         | string | Computed. The Firestore document ID `policy.opteryx` assigns. |
| `created_at` | string | Computed. |
| `updated_at` | string | Computed. |
| `updated_by` | string | Computed. |

Reserved patterns (`public.*`, `personal.*`, `*.information_schema.*`) are rejected by the API regardless of what the provider sends — see `app/models/policy.py::validate_pattern_does_not_target_reserved_resource` in `policy.opteryx`.

### Import

```bash
terraform import opteryx_access_policy.analytics_reader analytics/3f2b1a9e-1234-4c56-9abc-0123456789ab
```

The import ID is `workspace/policy_id` — a policy ID alone doesn't say which workspace's Firestore subcollection to look in.

## Data source: `opteryx_access_policies`

```hcl
data "opteryx_access_policies" "analytics" {
  workspace = "analytics"
}

output "analytics_grants" {
  value = data.opteryx_access_policies.analytics.policies
}
```

Returns `policies`, a list of `{ id, principal, role, pattern }` objects — one per policy document in the workspace. Useful for auditing what's granted today, or for finding a policy's `id` to hand to `terraform import`.

## Testing

```bash
make test      # unit tests -- currently just compile-time checks, no unit tests yet
make testacc   # acceptance tests -- hits a REAL policy.opteryx instance, creates/destroys real policies
```

`testacc` requires:

| Env var | Purpose |
|---|---|
| `OPTERYX_TOKEN` | Bearer token for an owner/admin identity |
| `OPTERYX_TEST_WORKSPACE` | A workspace that identity governs |
| `OPTERYX_TEST_PRINCIPAL` | A *different* identity to grant policies to during the test — the token's own identity can't be a principal |

## Project layout

```
main.go                                # plugin entrypoint
internal/client/                       # hand-written HTTP client for the access-policy API
internal/provider/                     # provider + resource implementations (terraform-plugin-framework)
examples/                              # example .tf used to generate registry docs later
```

## Not yet done

- `tfplugindocs generate` for `docs/` once the schema stabilizes
- Release automation (GoReleaser + GitHub Actions) if/when this needs to reach the public registry — not required for private/local use (`dev_overrides` or a private registry work without HashiCorp's involvement)
