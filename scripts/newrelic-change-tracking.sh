#!/usr/bin/env bash
# Record a New Relic Change Tracking deployment marker via NerdGraph.
#
# Looks up the target entity (APM application) by name, then creates a
# deployment so you can correlate this rollout with metric changes in NR.
#
# Required env:
#   NEW_RELIC_USER_API_KEY   User key (NRAK-...). NOT the ingest license key.
#   NEW_RELIC_APP_NAME       Exact entity name, e.g. "PerfectCatStreaming (dev)".
#   VERSION                  Deployment version (we use the git short SHA).
#
# Optional env:
#   NEW_RELIC_REGION         US | EU            (default: US)
#   NEW_RELIC_ENTITY_DOMAIN  APM | BROWSER ...  (default: APM)
#   COMMIT_SHA               Full commit hash   (default: empty)
#   NR_DEPLOY_DESCRIPTION     Free text          (default: "Deployed via CI")
#   NR_DEPLOY_USER           Who deployed        (default: empty)
#
# Exit codes: 0 ok, 1 bad config / API error, 2 entity not found.
set -euo pipefail

: "${NEW_RELIC_USER_API_KEY:?set NEW_RELIC_USER_API_KEY (NRAK-... user key)}"
: "${NEW_RELIC_APP_NAME:?set NEW_RELIC_APP_NAME (exact NR entity name)}"
: "${VERSION:?set VERSION (e.g. git short sha)}"

REGION="${NEW_RELIC_REGION:-US}"
DOMAIN="${NEW_RELIC_ENTITY_DOMAIN:-APM}"
COMMIT_SHA="${COMMIT_SHA:-}"
DESCRIPTION="${NR_DEPLOY_DESCRIPTION:-Deployed via CI}"
DEPLOY_USER="${NR_DEPLOY_USER:-}"

case "$REGION" in
  US|us) ENDPOINT="https://api.newrelic.com/graphql" ;;
  EU|eu) ENDPOINT="https://api.eu.newrelic.com/graphql" ;;
  *) echo "ERROR: NEW_RELIC_REGION must be US or EU" >&2; exit 1 ;;
esac

for cmd in curl jq; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: $cmd not in PATH" >&2; exit 1; }
done

nerdgraph() {
  # $1 = GraphQL document. Returns the raw JSON response on stdout.
  local query="$1"
  curl -sS -X POST "$ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "API-Key: ${NEW_RELIC_USER_API_KEY}" \
    --data "$(jq -n --arg q "$query" '{query: $q}')"
}

# --- 1) resolve the entity GUID by name ---
search_query="{ actor { entitySearch(query: \"name = '${NEW_RELIC_APP_NAME}' AND domain = '${DOMAIN}' AND type = 'APPLICATION'\") { results { entities { guid name } } } } }"
search_resp="$(nerdgraph "$search_query")"

if echo "$search_resp" | jq -e '.errors' >/dev/null 2>&1; then
  echo "ERROR: NerdGraph entitySearch failed:" >&2
  echo "$search_resp" | jq '.errors' >&2
  exit 1
fi

guid="$(echo "$search_resp" | jq -r '.data.actor.entitySearch.results.entities[0].guid // empty')"
if [[ -z "$guid" ]]; then
  echo "WARN: no ${DOMAIN} entity named '${NEW_RELIC_APP_NAME}' found — skipping change tracking." >&2
  echo "      (The app must have reported to New Relic at least once.)" >&2
  exit 2
fi
echo "[change-tracking] entity '${NEW_RELIC_APP_NAME}' -> ${guid}"

# --- 2) create the deployment marker ---
mutation="mutation { changeTrackingCreateDeployment(deployment: { entityGuid: \"${guid}\", version: \"${VERSION}\", description: \"${DESCRIPTION}\", commit: \"${COMMIT_SHA}\", user: \"${DEPLOY_USER}\", deploymentType: BASIC }) { deploymentId entityGuid } }"
deploy_resp="$(nerdgraph "$mutation")"

if echo "$deploy_resp" | jq -e '.errors' >/dev/null 2>&1; then
  echo "ERROR: changeTrackingCreateDeployment failed:" >&2
  echo "$deploy_resp" | jq '.errors' >&2
  exit 1
fi

deployment_id="$(echo "$deploy_resp" | jq -r '.data.changeTrackingCreateDeployment.deploymentId // empty')"
echo "[change-tracking] ✅ recorded deployment version=${VERSION} id=${deployment_id:-unknown}"
