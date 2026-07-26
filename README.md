# terraform-provider-opteryx

[![build](https://github.com/mabel-dev/terraform-provider-opteryx/actions/workflows/build.yml/badge.svg)](https://github.com/mabel-dev/terraform-provider-opteryx/actions/workflows/build.yml)

A Terraform provider for managing [policy.opteryx](https://github.com/mabel-dev/policy.opteryx.app) access policies as code, instead of through ad-hoc API calls.

## Status

Scaffold. One resource is implemented:

- `opteryx_access_policy` — a single principal/role/pattern grant, backed by `policy.opteryx`'s `/v1/access/workspace/{workspace}/policies...` CRUD endpoints.

No data source yet (e.g. to enumerate existing policies via the list endpoint) — add one if a use case shows up.

## Requirements

- [Go](https://go.dev/) 1.22+
- [Terraform](https://developer.hashicorp.com/terraform) 1.0+ (for local testing with the CLI)

## Building

```bash
go mod tidy   # resolves and locks dependencies -- go.sum isn't checked in yet
make build
```

No Go installed locally? Push a branch or open a PR — [`.github/workflows/build.yml`](.github/workflows/build.yml) runs `go mod tidy`, `gofmt`, `go vet`, `go build`, and `go test` on every push and pull request against `main`. Its "Upload go.sum" step attaches the resolved `go.sum` as a run artifact, downloadable from the run's summary page, so you can commit it without ever running Go yourself.

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

## Project layout

```
main.go                                # plugin entrypoint
internal/client/                       # hand-written HTTP client for the access-policy API
internal/provider/                     # provider + resource implementations (terraform-plugin-framework)
examples/                              # example .tf used to generate registry docs later
```

## Not yet done

- `go.sum` — CI resolves it on every run (see above); commit the artifact it uploads to lock it in
- Acceptance tests (`internal/provider/*_test.go`, gated by `TF_ACC`)
- `tfplugindocs generate` for `docs/` once the schema stabilizes
- Release automation (GoReleaser + GitHub Actions) if/when this needs to reach the public registry — not required for private/local use (`dev_overrides` or a private registry work without HashiCorp's involvement)
