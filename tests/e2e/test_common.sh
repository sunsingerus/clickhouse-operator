#!/bin/bash

# Common library for test scripts. Source this file, do not execute it.
# Usage: source "${CUR_DIR}/test_common.sh"

COMMON_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# =============================================================================
# Variable defaults (all overridable via environment)
# =============================================================================

# Operator versioning
OPERATOR_VERSION="${OPERATOR_VERSION:-"dev"}"
OPERATOR_DOCKER_REPO="${OPERATOR_DOCKER_REPO:-"altinity/clickhouse-operator"}"
OPERATOR_IMAGE="${OPERATOR_IMAGE:-"${OPERATOR_DOCKER_REPO}:${OPERATOR_VERSION}"}"
METRICS_EXPORTER_DOCKER_REPO="${METRICS_EXPORTER_DOCKER_REPO:-"altinity/metrics-exporter"}"
METRICS_EXPORTER_IMAGE="${METRICS_EXPORTER_IMAGE:-"${METRICS_EXPORTER_DOCKER_REPO}:${OPERATOR_VERSION}"}"

# NOTE: IMAGE_PULL_POLICY is intentionally NOT set here.
# Test runners default to "Always" (CI), local scripts default to "IfNotPresent" (minikube).

# Test execution
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-"test"}"
OPERATOR_INSTALL="${OPERATOR_INSTALL:-"yes"}"
ONLY="${ONLY:-"*"}"
VERBOSITY="${VERBOSITY:-"2"}"
RUN_ALL="${RUN_ALL:-""}"
KUBECTL_MODE="${KUBECTL_MODE:-"apply"}"

# Minikube control
MINIKUBE_RESET="${MINIKUBE_RESET:-""}"
MINIKUBE_PRELOAD_IMAGES="${MINIKUBE_PRELOAD_IMAGES:-""}"

# =============================================================================
# Image lists for preloading into minikube
# =============================================================================

PRELOAD_IMAGES_OPERATOR=(
    "clickhouse/clickhouse-server:23.3"
    "clickhouse/clickhouse-server:23.8"
    "clickhouse/clickhouse-server:24.3"
    "clickhouse/clickhouse-server:24.8"
    "clickhouse/clickhouse-server:25.3"
    "clickhouse/clickhouse-server:latest"
    "altinity/clickhouse-server:24.8.14.10459.altinitystable"
    "docker.io/zookeeper:3.8.4"
)

PRELOAD_IMAGES_KEEPER=(
    "clickhouse/clickhouse-server:23.3"
    "clickhouse/clickhouse-server:23.8"
    "clickhouse/clickhouse-server:24.3"
    "clickhouse/clickhouse-server:24.8"
    "clickhouse/clickhouse-server:25.3"
    "clickhouse/clickhouse-server:latest"
    "altinity/clickhouse-server:24.8.14.10459.altinitystable"
    "docker.io/zookeeper:3.8.4"
)

PRELOAD_IMAGES_METRICS=(
    "clickhouse/clickhouse-server:23.3"
    "clickhouse/clickhouse-server:25.3"
    "clickhouse/clickhouse-server:latest"
)

# =============================================================================
# Functions
# =============================================================================

# Install Python dependencies needed by TestFlows
function common_install_pip_requirements() {
    pip3 install -r "${COMMON_DIR}/../image/requirements.txt"
}

# Convert RUN_ALL env var to --test-to-end flag.
# Usage: RUN_ALL_FLAG=$(common_convert_run_all)
function common_convert_run_all() {
    if [[ -n "${RUN_ALL}" ]]; then
        echo "--test-to-end"
    fi
}

# Export the standard set of env vars that regression.py / settings.py expects
function common_export_test_env() {
    export OPERATOR_NAMESPACE
    export OPERATOR_INSTALL
    export IMAGE_PULL_POLICY
}

# Reset minikube cluster if MINIKUBE_RESET is set
function common_minikube_reset() {
    if [[ -n "${MINIKUBE_RESET}" ]]; then
        SKIP_K9S="yes" "${COMMON_DIR}/run_minikube_reset.sh"
    fi
}

# Pull images and load them into minikube.
# Only runs if MINIKUBE_PRELOAD_IMAGES is set.
# Usage: common_preload_images "${PRELOAD_IMAGES_OPERATOR[@]}"
function common_preload_images() {
    if [[ -n "${MINIKUBE_PRELOAD_IMAGES}" ]]; then
        echo "pre-load images into minikube"
        for image in "$@"; do
            docker pull -q "${image}" && \
            echo "pushing ${image} to minikube" && \
            minikube image load "${image}" --overwrite=false --daemon=true
        done
        echo "images pre-loaded"
    fi
}

# Build operator + metrics-exporter docker images and load them into minikube
function common_build_and_load_images() {
    echo "Build" && \
    VERBOSITY="${VERBOSITY}" "${COMMON_DIR}/../../dev/image_build_all_dev.sh" && \
    echo "Load images" && \
    minikube image load "${OPERATOR_IMAGE}" && \
    minikube image load "${METRICS_EXPORTER_IMAGE}" && \
    echo "Images prepared"
}

# Run a test runner script with all env vars forwarded.
# Usage: common_run_test_script "run_tests_operator.sh"
function common_run_test_script() {
    local script="${1}"
    OPERATOR_DOCKER_REPO="${OPERATOR_DOCKER_REPO}" \
    METRICS_EXPORTER_DOCKER_REPO="${METRICS_EXPORTER_DOCKER_REPO}" \
    OPERATOR_VERSION="${OPERATOR_VERSION}" \
    IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY}" \
    OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE}" \
    OPERATOR_INSTALL="${OPERATOR_INSTALL}" \
    ONLY="${ONLY}" \
    KUBECTL_MODE="${KUBECTL_MODE}" \
    RUN_ALL="${RUN_ALL}" \
    "${COMMON_DIR}/${script}"
}
