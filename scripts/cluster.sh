#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER_NAME="cassem-it"
CLUSTER_DIR="${ROOT_DIR}/.cluster/${CLUSTER_NAME}"
NETWORK="${CLUSTER_NAME}"
PODMAN="podman"

DB_IMAGE="localhost/cassemdb:${CLUSTER_NAME}"
ADM_IMAGE="localhost/cassemadm:${CLUSTER_NAME}"
AGENT_IMAGE="localhost/cassemagent:${CLUSTER_NAME}"

DB_ENDPOINTS="127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023"
DB_PORTS=("2021" "2022" "2023")
ADM_PORT="20218"
AGENT_PORT="20219"

usage() {
  cat <<'EOF'
Usage: scripts/cluster.sh <command>

Commands:
  start     Build images and start full local cluster
  stop      Stop and remove cluster containers and network
  restart   Stop, then start
  status    Print cluster containers and ports
  logs      Print cluster logs
  clean     Stop cluster and remove generated artifacts
EOF
}

need_podman() {
  if ! command -v "${PODMAN}" >/dev/null 2>&1; then
    echo "podman not found; install Podman and retry" >&2
    exit 1
  fi
}

container_name() {
  printf '%s-%s' "${CLUSTER_NAME}" "$1"
}

port_free() {
  local port="$1"
  if lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "port ${port} is already in use" >&2
    return 1
  fi
}

check_ports() {
  for port in "${DB_PORTS[@]}" "${ADM_PORT}" "${AGENT_PORT}"; do
    port_free "${port}"
  done
}

build_binary() {
  local output="$1"
  local pkg="$2"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o "${ROOT_DIR}/${output}" \
    -ldflags "-s -X main.Version=test -X main.BuildTime=test -X main.GitHash=test" \
    "${pkg}"
}

build_images() {
  build_binary "cassemdb" "./cmd/cassemdb"
  build_binary "cassemadm" "./cmd/cassemadm"
  build_binary "cassemagent" "./cmd/cassemagent"

  "${PODMAN}" build -t "${DB_IMAGE}" -f "${ROOT_DIR}/.deploy/dockerfiles/cassemdb.Dockerfile" "${ROOT_DIR}"
  "${PODMAN}" build -t "${ADM_IMAGE}" -f "${ROOT_DIR}/.deploy/dockerfiles/cassemadm.Dockerfile" "${ROOT_DIR}"
  "${PODMAN}" build -t "${AGENT_IMAGE}" -f "${ROOT_DIR}/.deploy/dockerfiles/cassemagent.Dockerfile" "${ROOT_DIR}"
}

write_configs() {
  rm -rf "${CLUSTER_DIR}/configs"
  mkdir -p "${CLUSTER_DIR}/configs" "${CLUSTER_DIR}/storage"

  local raft_cluster="http://cassemdb1:3021,http://cassemdb2:3022,http://cassemdb3:3023"
  for idx in 1 2 3; do
    local name="cassemdb${idx}"
    mkdir -p "${CLUSTER_DIR}/configs/${name}" "${CLUSTER_DIR}/storage/${name}"
    cat > "${CLUSTER_DIR}/configs/${name}/cassemdb.toml" <<EOF
debug = false
addr = ":2021"
[bolt]
    db = "cassemdb.kv"
[raft]
    bind = "http://${name}:302${idx}"
    cluster = "${raft_cluster}"
    snapCount = 300
EOF
  done

  mkdir -p "${CLUSTER_DIR}/configs/cassemadm"
  cat > "${CLUSTER_DIR}/configs/cassemadm/cassemadm.toml" <<'EOF'
debug = false
cassemdb = [
    "cassemdb1:2021",
    "cassemdb2:2021",
    "cassemdb3:2021"
]
[http]
    addr = ":20218"
EOF

  mkdir -p "${CLUSTER_DIR}/configs/cassemagent"
  cat > "${CLUSTER_DIR}/configs/cassemagent/cassemagent.toml" <<'EOF'
debug = false
ttl = 30
renewInterval = 20
elementCacheSize = 1000
cassemdb = [
    "cassemdb1:2021",
    "cassemdb2:2021",
    "cassemdb3:2021"
]
[server]
    addr = ":20219"
EOF
}

ensure_network() {
  if ! "${PODMAN}" network exists "${NETWORK}"; then
    "${PODMAN}" network create "${NETWORK}"
  fi
}

run_db_node() {
  local idx="$1"
  local name="cassemdb${idx}"
  local host_port="${DB_PORTS[$((idx - 1))]}"
  "${PODMAN}" run -d \
    --name "$(container_name "${name}")" \
    --hostname "${name}" \
    --network "${NETWORK}" \
    --network-alias "${name}" \
    -p "127.0.0.1:${host_port}:2021" \
    -v "${CLUSTER_DIR}/configs/${name}:/app/cassemdb/configs:Z" \
    -v "${CLUSTER_DIR}/storage/${name}:/app/cassemdb/storage:Z" \
    "${DB_IMAGE}" \
    ./cassemdb \
      --conf ./configs/cassemdb.toml \
      --endpoint :2021 \
      --raft.cluster "http://cassemdb1:3021,http://cassemdb2:3022,http://cassemdb3:3023" \
      --raft.bind "http://${name}:302${idx}" \
      --storage ./storage
}

run_adm() {
  "${PODMAN}" run -d \
    --name "$(container_name "cassemadm")" \
    --network "${NETWORK}" \
    -p "127.0.0.1:${ADM_PORT}:20218" \
    -v "${CLUSTER_DIR}/configs/cassemadm:/app/cassemadm/configs:Z" \
    "${ADM_IMAGE}"
}

run_agent() {
  "${PODMAN}" run -d \
    --name "$(container_name "cassemagent")" \
    --network "${NETWORK}" \
    -p "127.0.0.1:${AGENT_PORT}:20219" \
    -v "${CLUSTER_DIR}/configs/cassemagent:/app/cassemagent/configs:Z" \
    "${AGENT_IMAGE}"
}

wait_tcp() {
  local port="$1"
  local deadline=$((SECONDS + 30))
  while (( SECONDS < deadline )); do
    if nc -z 127.0.0.1 "${port}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "port ${port} did not become ready" >&2
  return 1
}

wait_ready() {
  for port in "${DB_PORTS[@]}" "${ADM_PORT}" "${AGENT_PORT}"; do
    wait_tcp "${port}"
  done
  (cd "${ROOT_DIR}" && go run ./scripts/cluster_healthcheck.go -endpoints "${DB_ENDPOINTS}" -timeout 45s)
}

print_status() {
  "${PODMAN}" ps -a --filter "name=${CLUSTER_NAME}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || true
  cat <<EOF

Endpoints:
  cassemdb: ${DB_ENDPOINTS}
  cassemadm: http://127.0.0.1:${ADM_PORT}
  cassemagent: 127.0.0.1:${AGENT_PORT}
EOF
}

print_logs() {
  for name in cassemdb1 cassemdb2 cassemdb3 cassemadm cassemagent; do
    local container
    container="$(container_name "${name}")"
    echo "===== ${container} ====="
    "${PODMAN}" logs "${container}" 2>&1 || true
  done
}

remove_container() {
  local name
  name="$(container_name "$1")"
  if "${PODMAN}" container exists "${name}"; then
    "${PODMAN}" rm -f "${name}" >/dev/null
  fi
}

stop_cluster() {
  need_podman
  for name in cassemagent cassemadm cassemdb3 cassemdb2 cassemdb1; do
    remove_container "${name}"
  done
  if "${PODMAN}" network exists "${NETWORK}"; then
    "${PODMAN}" network rm "${NETWORK}" >/dev/null || true
  fi
}

start_cluster() {
  need_podman
  stop_cluster
  check_ports
  build_images
  write_configs
  ensure_network
  run_db_node 1
  run_db_node 2
  run_db_node 3
  run_adm
  run_agent
  if ! wait_ready; then
    print_status
    print_logs
    exit 1
  fi
  print_status
}

clean_cluster() {
  stop_cluster
  rm -rf "${CLUSTER_DIR}"
  rm -f "${ROOT_DIR}/cassemdb" "${ROOT_DIR}/cassemadm" "${ROOT_DIR}/cassemagent"
}

cmd="${1:-}"
case "${cmd}" in
  start)
    start_cluster
    ;;
  stop)
    stop_cluster
    ;;
  restart)
    stop_cluster
    start_cluster
    ;;
  status)
    need_podman
    print_status
    ;;
  logs)
    need_podman
    print_logs
    ;;
  clean)
    clean_cluster
    ;;
  *)
    usage
    exit 2
    ;;
esac
