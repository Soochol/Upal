#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[dev]${NC} $*"; }
warn() { echo -e "${YELLOW}[dev]${NC} $*"; }
fail() { echo -e "${RED}[dev]${NC} $*"; }

kill_port() {
  local port=$1
  local pids
  pids=$(lsof -ti :"$port" 2>/dev/null || true)
  if [ -n "$pids" ]; then
    echo "$pids" | xargs kill 2>/dev/null || true
    sleep 1
  fi
}

wait_for_port() {
  local port=$1 name=$2 max=20
  for i in $(seq 1 $max); do
    if lsof -ti :"$port" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

mkdir -p tmp

# --- 1. Database ---
log "Starting PostgreSQL..."
if docker compose ps postgres --format '{{.State}}' 2>/dev/null | grep -q running; then
  docker compose restart postgres >/dev/null 2>&1
else
  docker compose up -d postgres >/dev/null 2>&1
fi

if ! docker compose exec postgres pg_isready -U upal >/dev/null 2>&1; then
  for i in $(seq 1 10); do
    sleep 1
    docker compose exec postgres pg_isready -U upal >/dev/null 2>&1 && break
  done
fi

# --- 2. Backend ---
log "Starting backend (air → :8081)..."
kill_port 8081
nohup air > tmp/backend.log 2>&1 &

# --- 3. Frontend ---
log "Starting frontend (vite → :5173)..."
kill_port 5173
(cd web && nohup npm run dev > ../tmp/frontend.log 2>&1 &)

# --- 4. Verify ---
log "Waiting for services..."

db_ok=false; be_ok=false; fe_ok=false

docker compose exec postgres pg_isready -U upal >/dev/null 2>&1 && db_ok=true
wait_for_port 8081 "backend"  && be_ok=true
wait_for_port 5173 "frontend" && fe_ok=true

echo ""
echo "┌────────────┬───────┬────────┐"
echo "│ Service    │ Port  │ Status │"
echo "├────────────┼───────┼────────┤"
$db_ok && echo -e "│ PostgreSQL │ 5432  │ ${GREEN}UP${NC}     │" || echo -e "│ PostgreSQL │ 5432  │ ${RED}DOWN${NC}   │"
$be_ok && echo -e "│ Backend    │ 8081  │ ${GREEN}UP${NC}     │" || echo -e "│ Backend    │ 8081  │ ${RED}DOWN${NC}   │"
$fe_ok && echo -e "│ Frontend   │ 5173  │ ${GREEN}UP${NC}     │" || echo -e "│ Frontend   │ 5173  │ ${RED}DOWN${NC}   │"
echo "└────────────┴───────┴────────┘"
echo ""
$fe_ok && log "http://localhost:5173"
