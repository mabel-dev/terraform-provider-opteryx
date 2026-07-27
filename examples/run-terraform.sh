#!/bin/sh
# Wrapper that mints a fresh OPTERYX_TOKEN immediately before invoking
# terraform. Necessary because authenticate.opteryx issues access tokens with
# only a 5-minute lifetime (ACCESS_TOKEN_EXPIRATION_MINUTES in core.py) -- a
# token exported earlier in a long session can expire mid-apply.
#
# Requires OPTERYX_CLIENT_ID / OPTERYX_CLIENT_SECRET for a client-credentials
# identity created via authenticate.opteryx's POST /clients/{client_id}/credentials.
# That identity must be an owner/admin on the workspaces you manage, and must
# NOT appear as a `principal` in any opteryx_access_policy this config
# manages -- policy.opteryx rejects self-grants.
set -eu

: "${OPTERYX_AUTH_URL:=https://authenticate.opteryx.app}"

OPTERYX_TOKEN=$(curl -sf -X POST "$OPTERYX_AUTH_URL/token" \
  -d grant_type=client_credentials \
  -d client_id="$OPTERYX_CLIENT_ID" \
  -d client_secret="$OPTERYX_CLIENT_SECRET" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])')
export OPTERYX_TOKEN

exec terraform "$@"
