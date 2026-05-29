#!/usr/bin/env bash
set -euo pipefail

REGISTRY_NAME="kind-registry"
REGISTRY_PORT="${REGISTRY_PORT:-5001}"
CLUSTER_NAME="${CLUSTER_NAME:-arcana-dev}"

CONTAINER_CMD="${CONTAINER_CMD:-$(command -v podman 2>/dev/null || command -v docker)}"

if ${CONTAINER_CMD} inspect "${REGISTRY_NAME}" &>/dev/null; then
  echo "Registry '${REGISTRY_NAME}' already running."
else
  echo "Starting local registry on port ${REGISTRY_PORT}..."
  ${CONTAINER_CMD} run -d --restart=always \
    -p "127.0.0.1:${REGISTRY_PORT}:5000" \
    --network bridge \
    --name "${REGISTRY_NAME}" \
    registry:2
fi

if [ "$(${CONTAINER_CMD} inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REGISTRY_NAME}" 2>/dev/null)" = 'null' ] 2>/dev/null; then
  ${CONTAINER_CMD} network connect "kind" "${REGISTRY_NAME}" 2>/dev/null || true
fi

echo "Registry available at localhost:${REGISTRY_PORT}"
