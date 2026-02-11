#!/bin/bash
CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${CUR_DIR}/test_common.sh"

# Test component select options:
# - operator
# - keeper
# - metrics
# Can be set via env var for non-interactive use: WHAT=metrics ./run_tests_local.sh
WHAT="${WHAT}"

# Repeat mode options:
# - success = repeat until success
# - failure = repeat until failure
# - not specified / empty = single run
# Usage: REPEAT_UNTIL=success ./run_tests_local.sh
REPEAT_UNTIL="${REPEAT_UNTIL:-""}"

#
# Interactive menu (or non-interactive if WHAT is already set)
#
function select_test_goal() {
    local specified_goal="${1}"
    if [[ -n "${specified_goal}" ]]; then
        echo "Having specified explicitly: ${specified_goal}" >&2
        echo "${specified_goal}"
        return
    fi

    echo "What would you like to start? Possible options:" >&2
    echo "  1     - test operator" >&2
    echo "  2     - test keeper" >&2
    echo "  3     - test metrics" >&2
    echo -n "Enter your choice (1, 2, 3): " >&2
    read COMMAND
    COMMAND=$(echo "${COMMAND}" | tr -d '\n\t\r ')
    case "${COMMAND}" in
        "1") echo "operator" ;;
        "2") echo "keeper" ;;
        "3") echo "metrics" ;;
        *)
            echo "don't know what '${COMMAND}' is, so picking operator" >&2
            echo "operator"
            ;;
    esac
}

WHAT=$(select_test_goal "${WHAT}")

# Map test goal to dedicated local script
case "${WHAT}" in
    "operator")
        LOCAL_SCRIPT="run_tests_operator_local.sh"
        echo "Selected: test OPERATOR"
        ;;
    "keeper")
        LOCAL_SCRIPT="run_tests_keeper_local.sh"
        echo "Selected: test KEEPER"
        ;;
    "metrics")
        LOCAL_SCRIPT="run_tests_metrics_local.sh"
        echo "Selected: test METRICS"
        ;;
    *)
        echo "Unknown test type: '${WHAT}', exiting"
        exit 1
        ;;
esac

TIMEOUT=30
echo "Press <ENTER> to start test immediately (if you agree with specified options)"
echo "In case no input provided tests would start in ${TIMEOUT} seconds automatically"
read -t ${TIMEOUT}

# Dispatch to the dedicated local script, with optional repeat mode
case "${REPEAT_UNTIL}" in
    "success")
        # Repeat until tests pass
        start=$(date)
        run=1
        echo "start run ${run}"
        until "${CUR_DIR}/${LOCAL_SCRIPT}"; do
            echo "run number ${run} failed"
            echo "-------------------------------------------"
            run=$((run+1))
            echo "start run ${run}"
        done
        end=$(date)
        echo "============================================="
        echo "Run number ${run} succeeded"
        echo "start time: ${start}"
        echo "end   time: ${end}"
        ;;
    "failure")
        # Repeat until tests fail
        start=$(date)
        run=1
        echo "start run ${run}"
        while "${CUR_DIR}/${LOCAL_SCRIPT}"; do
            echo "run number ${run} completed successfully"
            echo "-------------------------------------------"
            run=$((run+1))
            echo "start run ${run}"
        done
        end=$(date)
        echo "============================================="
        echo "Run number ${run} failed"
        echo "start time: ${start}"
        echo "end   time: ${end}"
        ;;
    *)
        # Single run
        "${CUR_DIR}/${LOCAL_SCRIPT}"
        ;;
esac
