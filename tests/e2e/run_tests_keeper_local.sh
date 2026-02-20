#!/bin/bash
CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${CUR_DIR}/test_common.sh"

IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-"IfNotPresent"}"

common_minikube_reset
common_preload_images "${PRELOAD_IMAGES_KEEPER[@]}"
common_build_and_load_images && \
common_run_test_script "run_tests_keeper.sh"
