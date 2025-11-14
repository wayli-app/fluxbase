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

# Print header with proper padding
# Box width is 59 characters (61 including ║)
# Calculate padding to center the text
TEXT="🧪 Running $TEST_TYPE Test Suite"
TEXT_LENGTH=${#TEXT}
# Account for emoji taking 2 display columns but 4 bytes
DISPLAY_LENGTH=$((TEXT_LENGTH - 2))
PADDING=$(( (59 - DISPLAY_LENGTH) / 2 ))
LEFT_PAD=$(printf '%*s' $PADDING '')
RIGHT_PAD=$(printf '%*s' $((59 - DISPLAY_LENGTH - PADDING)) '')

echo ""
echo -e "${BOLD}${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}${CYAN}║${LEFT_PAD}${TEXT}${RIGHT_PAD}║${NC}"
echo -e "${BOLD}${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Run tests and capture output
START_TIME=$(date +%s)

# Add -json flag to the test command for structured output
TEST_CMD_JSON="${TEST_CMD/go test/go test -json}"

# Parse JSON output to show real-time progress
eval "$TEST_CMD_JSON" 2>&1 | tee "$TEST_OUTPUT" | awk '
BEGIN {
    running = 0
    passed = 0
    failed = 0
}
/^{/ {
    # Parse JSON line - extract test name if present
    if (match($0, /"Test":"([^"]+)"/)) {
        test = substr($0, RSTART+8, RLENGTH-9)
    } else {
        test = ""
    }

    # Check action type
    if (match($0, /"Action":"run"/)) {
        if (test != "") {
            printf "\r\033[K\033[0;36m▶\033[0m Running: \033[1m%s\033[0m", test
            fflush()
            running++
        }
    }
    else if (match($0, /"Action":"pass"/)) {
        if (test != "" && index(test, "/") == 0) {
            passed++
            printf "\r\033[K\033[0;32m✓\033[0m \033[2m%s\033[0m\n", test
            fflush()
        }
    }
    else if (match($0, /"Action":"fail"/)) {
        if (test != "" && index(test, "/") == 0) {
            failed++
            printf "\r\033[K\033[0;31m✗\033[0m \033[1m%s\033[0m\n", test
            fflush()
        }
    }
}
!/^{/ {
    # Non-JSON lines - only show non-verbose logs
    if ($0 !~ /^\{"level":/ && $0 !~ /^\[/) {
        print $0
    }
}
END {
    printf "\r\033[K"
}
'

TEST_EXIT_CODE=${PIPESTATUS[0]}
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Extract test results from JSON output
# Use grep -c with proper error handling and ensure we get a valid number
PASSED=$(grep -o '"Action":"pass"' "$TEST_OUTPUT" 2>/dev/null | grep -c . || echo 0)
FAILED=$(grep -o '"Action":"fail"' "$TEST_OUTPUT" 2>/dev/null | grep -c . || echo 0)

# Count only top-level tests (exclude subtests)
PASSED=$(awk '/"Action":"pass"/ && /"Test":"[^"]*"/ && !/"Test":"[^"]*\//' "$TEST_OUTPUT" 2>/dev/null | wc -l | tr -d ' ')
FAILED=$(awk '/"Action":"fail"/ && /"Test":"[^"]*"/ && !/"Test":"[^"]*\//' "$TEST_OUTPUT" 2>/dev/null | wc -l | tr -d ' ')

# Ensure PASSED and FAILED are valid integers
if [ -z "$PASSED" ] || ! [[ "$PASSED" =~ ^[0-9]+$ ]]; then
    PASSED=0
fi
if [ -z "$FAILED" ] || ! [[ "$FAILED" =~ ^[0-9]+$ ]]; then
    FAILED=0
fi

TOTAL=$((PASSED + FAILED))

# Extract failed test names from JSON output
grep '"Action":"fail"' "$TEST_OUTPUT" | \
  grep '"Test":"' | \
  grep -v '"Test":"[^"]*/' | \
  sed 's/.*"Test":"\([^"]*\)".*/\1/' > "$FAILED_TESTS"

# Extract failure details for each failed test from JSON output
while IFS= read -r test_name; do
    if [ -n "$test_name" ]; then
        # Find the output messages for this test using grep/sed instead of AWK capture groups
        grep "\"Test\":\"$test_name\"" "$TEST_OUTPUT" | \
          grep '"Action":"output"' | \
          sed 's/.*"Output":"\([^"]*\)".*/\1/' | \
          grep -E "(Error:|FAIL:)" | \
          sed 's/\\n/\n/g; s/\\t/    /g' >> "$FAILURE_DETAILS"
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

# Show failed tests overview if any
if [ "$FAILED" -gt 0 ]; then
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}              FAILED TESTS OVERVIEW${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo ""

    # Show all failed test names with numbers
    awk 'NF {printf "  \033[0;31m%d.\033[0m \033[1m%s\033[0m\n", NR, $0}' "$FAILED_TESTS" | head -20

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
    TEST_NUM=1
    while IFS= read -r test_name; do
        if [ -n "$test_name" ] && [ "$SHOWN" -lt 3 ]; then
            echo -e "${BOLD}${RED}$TEST_NUM.${NC} ${BOLD}${YELLOW}$test_name${NC}"
            echo ""

            # Extract and show the error for this specific test from JSON
            # Get all output lines for this test, decode JSON escapes, and show relevant error context
            ALL_TEST_OUTPUT=$(grep "\"Test\":\"$test_name\"" "$TEST_OUTPUT" | \
                grep '"Action":"output"' | \
                sed 's/.*"Output":"//; s/"[,}].*//; s/\\n/\n/g; s/\\t/    /g; s/\\"/"/g')

            # Try to find error-related output
            ERROR_OUTPUT=$(echo "$ALL_TEST_OUTPUT" | awk '
                /Error:|FAIL:|Test:|Error Trace:|Messages:|panic/ { printing=1 }
                printing { print; lines++ }
                lines >= 30 { exit }
            ')

            if [ -n "$ERROR_OUTPUT" ]; then
                echo "$ERROR_OUTPUT" | sed 's/^/  /'
            else
                # If no error patterns found, show last 20 lines of test output
                LAST_OUTPUT=$(echo "$ALL_TEST_OUTPUT" | grep -v '^{"level"' | tail -20)
                if [ -n "$LAST_OUTPUT" ]; then
                    echo -e "  ${YELLOW}Last output from test:${NC}"
                    echo "$LAST_OUTPUT" | sed 's/^/  /'
                else
                    echo -e "  ${YELLOW}No error details available - test may have timed out or panicked${NC}"
                fi
            fi

            echo ""
            SHOWN=$((SHOWN + 1))
        fi
        TEST_NUM=$((TEST_NUM + 1))
    done < "$FAILED_TESTS"

    if [ "$FAILED" -gt 3 ]; then
        echo -e "${YELLOW}... and details for $((FAILED - 3)) more failed tests${NC}"
        echo -e "${YELLOW}Run tests individually with: go test -v -run <TestName>${NC}"
        echo ""
    fi
fi

echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Exit with the same code as the test command
exit $TEST_EXIT_CODE
