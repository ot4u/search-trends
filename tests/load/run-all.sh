set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

echo "Running read-only load scenario"
bash tests/load/read-10k.sh

echo
echo "Running cache-hit load scenario"
bash tests/load/cache-hit.sh

echo
echo "Running mixed read/write load scenario"
bash tests/load/mixed.sh
