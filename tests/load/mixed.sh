set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
source "${ROOT_DIR}/tests/load/common.sh"
cd "${ROOT_DIR}"

BASE_URL="${BASE_URL:-http://localhost:8080}"
NATS_URL="${NATS_URL:-nats://localhost:4222}"
SUBJECT="${SUBJECT:-search.events}"
WARMUP_EVENTS="${WARMUP_EVENTS:-20000}"
READ_RATE="${READ_RATE:-5000}"
WRITE_RATE="${WRITE_RATE:-1000}"
DURATION="${DURATION:-30s}"

seed_nats "${NATS_URL}" "${SUBJECT}" "${WARMUP_EVENTS}"
wait_for_trending_data "${BASE_URL}" 30

go run ./tests/load/publisher \
    --nats-url="${NATS_URL}" \
    --subject="${SUBJECT}" \
    --rate="${WRITE_RATE}" \
    --duration="${DURATION}" \
  >/dev/null 2>&1 &
PUBLISHER_PID=$!

sed "s|http://127.0.0.1:8080|${BASE_URL}|g" tests/load/read.targets \
  | vegeta_attack "${READ_RATE}" "${DURATION}" \
  | vegeta report

wait "${PUBLISHER_PID}"
