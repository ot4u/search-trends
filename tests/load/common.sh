set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

ensure_vegeta() {
  if command -v go >/dev/null 2>&1; then
    local gobin
    gobin="$(go env GOPATH 2>/dev/null)/bin"
    if [[ -n "${gobin}" && -d "${gobin}" ]]; then
      case ":${PATH}:" in
        *":${gobin}:"*) ;;
        *) export PATH="${PATH}:${gobin}" ;;
      esac
    fi
  fi

  if ! command -v vegeta >/dev/null 2>&1; then
    echo "vegeta not found in PATH. Install with:" >&2
    echo "  go install github.com/tsenart/vegeta/v12@latest" >&2
    exit 127
  fi
}

ensure_vegeta

vegeta_attack() {
  local rate="$1"
  local duration="$2"
  shift 2

  vegeta attack \
    -rate="${rate}" \
    -duration="${duration}" \
    -max-workers="${MAX_WORKERS:-2000}" \
    -connections="${MAX_CONNECTIONS:-2000}" \
    "$@"
}

seed_nats() {
  local nats_url="$1"
  local subject="$2"
  local count="$3"

  echo "Seeding ${count} events into ${subject} via ${nats_url}"
  (
    cd "${ROOT_DIR}"
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
