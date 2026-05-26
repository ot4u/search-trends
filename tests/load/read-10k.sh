set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
source "${ROOT_DIR}/tests/load/common.sh"
cd "${ROOT_DIR}"

BASE_URL="${BASE_URL:-http://localhost:8080}"
NATS_URL="${NATS_URL:-nats://localhost:4222}"
SUBJECT="${SUBJECT:-search.events}"
SEED_EVENTS="${SEED_EVENTS:-50000}"
RATE="${RATE:-10000}"
DURATION="${DURATION:-30s}"

seed_nats "${NATS_URL}" "${SUBJECT}" "${SEED_EVENTS}"
wait_for_trending_data "${BASE_URL}" 30

sed "s|http://127.0.0.1:8080|${BASE_URL}|g" tests/load/read.targets \
  | vegeta_attack "${RATE}" "${DURATION}" \
  | vegeta report
