set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

seed_nats() {
  local nats_url="$1"
  local subject="$2"
  local count="$3"

  echo "Seeding ${count} events into ${subject} via ${nats_url}"
  (
    cd "${ROOT_DIR}"
    GOCACHE="${ROOT_DIR}/.cache/go-build" GOMODCACHE="${ROOT_DIR}/.cache/go-mod" \
      go run ./tests/load/publisher \
        --nats-url="${nats_url}" \
        --subject="${subject}" \
        --count="${count}"
  )
}

wait_for_trending_data() {
  local base_url="$1"
  local timeout_seconds="${2:-30}"
  local deadline=$((SECONDS + timeout_seconds))

  echo "Waiting for trending snapshot to contain data"

  while (( SECONDS < deadline )); do
    if curl -fsS "${base_url}/api/v1/trending?limit=10" | grep -q '"query":'; then
      echo "Trending snapshot is ready"
      return 0
    fi
    sleep 1
  done

  echo "Timed out waiting for trending snapshot readiness" >&2
  return 1
}
