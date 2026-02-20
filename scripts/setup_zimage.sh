#!/usr/bin/env bash
# setup_zimage.sh — Set up Z-IMAGE inference environment for Upal
#
# Usage:
#   ./scripts/setup_zimage.sh              # Full install (torch + diffusers + model download)
#   ./scripts/setup_zimage.sh --mock-only  # Mock mode only (no GPU needed)
#
# The script creates a Python venv at .venv-zimage/ and installs dependencies.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
VENV_DIR="${PROJECT_DIR}/.venv-zimage"
MODEL_ID="Tongyi-MAI/Z-Image"
MOCK_ONLY=false

for arg in "$@"; do
    case "$arg" in
        --mock-only) MOCK_ONLY=true ;;
        *) echo "Unknown argument: $arg"; exit 1 ;;
    esac
done

echo "=== Z-IMAGE Setup ==="
echo "Project: ${PROJECT_DIR}"
echo "Venv:    ${VENV_DIR}"
echo "Mode:    $(if $MOCK_ONLY; then echo 'mock-only'; else echo 'full'; fi)"
echo ""

# Create venv if needed
if [ ! -d "$VENV_DIR" ]; then
    echo "Creating Python venv..."
    python3 -m venv "$VENV_DIR"
fi

# Activate
source "${VENV_DIR}/bin/activate"
pip install --upgrade pip -q

if $MOCK_ONLY; then
    echo "Installing mock-mode dependencies..."
    pip install fastapi "uvicorn[standard]" Pillow pydantic -q
    echo ""
    echo "=== Done (mock-only) ==="
    echo ""
    echo "To start mock server:"
    echo "  source ${VENV_DIR}/bin/activate"
    echo "  python scripts/zimage_server.py --mock --port 8090"
else
    echo "Installing full dependencies..."
    pip install -r "${SCRIPT_DIR}/requirements-zimage.txt" -q
    echo ""

    MODEL_DIR="${PROJECT_DIR}/models/z-image"
    echo "Downloading model to: ${MODEL_DIR}..."
    python3 -c "
from huggingface_hub import snapshot_download
snapshot_download('${MODEL_ID}', local_dir='${MODEL_DIR}')
print('Model downloaded successfully.')
"
    echo ""
    echo "=== Done (full) ==="
    echo ""
    echo "To start server:"
    echo "  source ${VENV_DIR}/bin/activate"
    echo "  python scripts/zimage_server.py --port 8090"
    echo ""
    echo "To run self-test (requires GPU):"
    echo "  python scripts/zimage_server.py --self-test"
fi
