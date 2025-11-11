#!/bin/bash
# Test runner script that provides detailed test summaries
# Used by Makefile test targets

set -o pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Temporary files
TEST_OUTPUT=$(mktemp)
FAILED_TESTS=$(mktemp)
FAILURE_DETAILS=$(mktemp)

# Cleanup on exit
cleanup() {
    rm -f "$TEST_OUTPUT" "$FAILED_TESTS" "$FAILURE_DETAILS"
}
trap cleanup EXIT

# Parse command line arguments
TEST_CMD="$@"
TEST_TYPE="E2E"

if [[ "$TEST_CMD" == *"./test/e2e/"* ]]; then
    TEST_TYPE="E2E"
elif [[ "$TEST_CMD" == *"-short"* ]]; then
    TEST_TYPE="Unit"
else
    TEST_TYPE="Integration"
fi

# Print header
echo ""
echo -e "${BOLD}${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}${CYAN}║         🧪 Running $TEST_TYPE Test Suite                    ║${NC}"
echo -e "${BOLD}${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Run tests and capture output
START_TIME=$(date +%s)
eval "$TEST_CMD" 2>&1 | tee "$TEST_OUTPUT"
TEST_EXIT_CODE=${PIPESTATUS[0]}
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Extract test results
# Use grep -c with proper error handling and ensure we get a valid number
PASSED=$(grep "^--- PASS:" "$TEST_OUTPUT" 2>/dev/null | wc -l | tr -d ' ')
FAILED=$(grep "^--- FAIL:" "$TEST_OUTPUT" 2>/dev/null | wc -l | tr -d ' ')

# Ensure PASSED and FAILED are valid integers
if [ -z "$PASSED" ] || ! [[ "$PASSED" =~ ^[0-9]+$ ]]; then
    PASSED=0
fi
if [ -z "$FAILED" ] || ! [[ "$FAILED" =~ ^[0-9]+$ ]]; then
    FAILED=0
fi

TOTAL=$((PASSED + FAILED))

# Extract failed test names
grep "^--- FAIL:" "$TEST_OUTPUT" | awk '{print $3}' | sed 's/(.*//' > "$FAILED_TESTS"

# Extract failure details for each failed test
while IFS= read -r test_name; do
    if [ -n "$test_name" ]; then
        # Find the failure message for this test
        awk "/^--- FAIL: $test_name/,/^(---|\t)/" "$TEST_OUTPUT" |
            grep -A 3 "Error:" |
            head -4 >> "$FAILURE_DETAILS"
        echo "---" >> "$FAILURE_DETAILS"
    fi
done < "$FAILED_TESTS"

# Calculate success rate
if [ "$TOTAL" -gt 0 ]; then
    SUCCESS_RATE=$((PASSED * 100 / TOTAL))
else
    SUCCESS_RATE=0
fi

# Print summary
echo ""
echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}${CYAN}              TEST RESULTS SUMMARY${NC}"
echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${GREEN}✅ PASSED:${NC}       ${BOLD}$PASSED${NC} tests"
echo -e "  ${RED}❌ FAILED:${NC}       ${BOLD}$FAILED${NC} tests"
echo -e "  ${BLUE}📊 TOTAL:${NC}        ${BOLD}$TOTAL${NC} tests"
echo -e "  ${CYAN}⏱️  DURATION:${NC}     ${BOLD}${DURATION}s${NC}"

if [ "$TOTAL" -gt 0 ]; then
    if [ "$SUCCESS_RATE" -ge 90 ]; then
        echo -e "  ${GREEN}📈 SUCCESS RATE:${NC} ${BOLD}${SUCCESS_RATE}%${NC}"
    elif [ "$SUCCESS_RATE" -ge 70 ]; then
        echo -e "  ${YELLOW}📈 SUCCESS RATE:${NC} ${BOLD}${SUCCESS_RATE}%${NC}"
    else
        echo -e "  ${RED}📈 SUCCESS RATE:${NC} ${BOLD}${SUCCESS_RATE}%${NC}"
    fi
fi

echo ""

# Show enhanced RLS test status if this is an E2E test run
if [ "$TEST_TYPE" = "E2E" ]; then
    RLS_PASSED=$(grep "^--- PASS:" "$TEST_OUTPUT" | grep -E "RLSAuth|RLSDashboard|RLSImpersonation|RLSAPI|RLSForce|RLSPerformance|RLSToken|RLSWebhook" | wc -l | tr -d ' ')
    RLS_FAILED=$(grep "^--- FAIL:" "$TEST_OUTPUT" | grep -E "RLSAuth|RLSDashboard|RLSImpersonation|RLSAPI|RLSForce|RLSPerformance|RLSToken|RLSWebhook" | wc -l | tr -d ' ')

    # Ensure RLS counts are valid integers
    if [ -z "$RLS_PASSED" ] || ! [[ "$RLS_PASSED" =~ ^[0-9]+$ ]]; then
        RLS_PASSED=0
    fi
    if [ -z "$RLS_FAILED" ] || ! [[ "$RLS_FAILED" =~ ^[0-9]+$ ]]; then
        RLS_FAILED=0
    fi

    RLS_TOTAL=$((RLS_PASSED + RLS_FAILED))

    if [ "$RLS_TOTAL" -gt 0 ]; then
        echo -e "${BOLD}${CYAN}Enhanced RLS Policy Tests:${NC}"
        echo -e "  ${GREEN}✅${NC} $RLS_PASSED / $RLS_TOTAL enhanced RLS tests passing"
        echo ""
    fi
fi

# Show failed tests if any
if [ "$FAILED" -gt 0 ]; then
    echo -e "${BOLD}${RED}Failed Tests ($FAILED):${NC}"
    echo ""

    cat "$FAILED_TESTS" | head -20 | while IFS= read -r test_name; do
        if [ -n "$test_name" ]; then
            echo -e "  ${RED}❌${NC} $test_name"
        fi
    done

    if [ "$FAILED" -gt 20 ]; then
        echo -e "  ${YELLOW}... and $((FAILED - 20)) more${NC}"
    fi

    echo ""
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}              FAILURE DETAILS${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo ""

    # Show detailed failure information for first few tests
    SHOWN=0
    while IFS= read -r test_name; do
        if [ -n "$test_name" ] && [ "$SHOWN" -lt 5 ]; then
            echo -e "${BOLD}${YELLOW}Test:${NC} $test_name"

            # Extract and show the error for this specific test
            awk "/^--- FAIL: $test_name/,/^---/" "$TEST_OUTPUT" | \
                grep -A 5 "Error:" | \
                head -6 | \
                sed 's/^/  /' | \
                sed "s/Error:/${RED}Error:${NC}/" | \
                sed "s/expected:/${CYAN}expected:${NC}/" | \
                sed "s/actual:/${CYAN}actual:${NC}/"

            echo ""
            SHOWN=$((SHOWN + 1))
        fi
    done < "$FAILED_TESTS"

    if [ "$FAILED" -gt 5 ]; then
        echo -e "${YELLOW}... and details for $((FAILED - 5)) more failed tests${NC}"
        echo -e "${YELLOW}Run with -v flag for full output${NC}"
        echo ""
    fi
fi

echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Exit with the same code as the test command
exit $TEST_EXIT_CODE
