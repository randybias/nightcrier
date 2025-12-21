#!/usr/bin/env bash
#
# validate-harness.sh - Comprehensive validation of test harness
#
# This script performs dry-run validation of the test harness without requiring
# live cluster resources, nightcrier binary, or secrets. It validates:
# - Script syntax and structure
# - Argument parsing and validation
# - Config generation logic
# - Error handling paths
# - Integration point wiring
#
# Usage: ./validate-harness.sh [--verbose]
#

set -euo pipefail

# Script paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TESTS_DIR="${SCRIPT_DIR}"
LIB_DIR="${TESTS_DIR}/lib"
FAILURE_DIR="${TESTS_DIR}/failure-induction"

# Validation state
VERBOSE=false
ERRORS=0
WARNINGS=0
CHECKS=0

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    printf "${BLUE}[INFO]${NC} %s\n" "$*"
}

log_success() {
    printf "${GREEN}[PASS]${NC} %s\n" "$*"
}

log_warning() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$*"
    ((WARNINGS++))
}

log_error() {
    printf "${RED}[FAIL]${NC} %s\n" "$*"
    ((ERRORS++))
}

log_check() {
    ((CHECKS++))
    if [ "$VERBOSE" = true ]; then
        printf "${BLUE}[CHECK ${CHECKS}]${NC} %s\n" "$*"
    fi
}

# Validate bash syntax for a script
validate_syntax() {
    local script="$1"
    local name
    name="$(basename "$script")"

    log_check "Validating syntax: $name"

    if [ ! -f "$script" ]; then
        log_error "Script not found: $script"
        return 1
    fi

    if ! bash -n "$script" 2>&1; then
        log_error "Syntax validation failed: $name"
        return 1
    fi

    log_success "Syntax valid: $name"
    return 0
}

# Check if script is executable
check_executable() {
    local script="$1"
    local name
    name="$(basename "$script")"

    log_check "Checking executable: $name"

    if [ ! -x "$script" ]; then
        log_warning "Script not executable: $name"
        return 1
    fi

    log_success "Executable: $name"
    return 0
}

# Validate shebang
validate_shebang() {
    local script="$1"
    local name
    name="$(basename "$script")"

    log_check "Validating shebang: $name"

    local shebang
    shebang="$(head -n 1 "$script")"

    if [[ ! "$shebang" =~ ^#!/usr/bin/env\ bash$ ]]; then
        log_warning "Incorrect shebang in $name: $shebang"
        return 1
    fi

    log_success "Shebang correct: $name"
    return 0
}

# Test argument parsing for main orchestrator
test_argument_parsing() {
    log_info "Testing argument parsing..."

    local main_script="${TESTS_DIR}/run-live-test.sh"

    # Test: No arguments (should show usage)
    log_check "Test: No arguments"
    if bash -n "$main_script" 2>&1; then
        log_success "Script loads without syntax errors"
    else
        log_error "Script has syntax errors"
        return 1
    fi

    # Validate that usage function exists
    log_check "Test: usage() function exists"
    if grep -q "^usage()" "$main_script"; then
        log_success "usage() function found"
    else
        log_error "usage() function not found"
        return 1
    fi

    # Validate agent validation exists
    log_check "Test: Agent validation logic exists"
    if grep -q "claude|codex|gemini" "$main_script"; then
        log_success "Agent validation logic found"
    else
        log_error "Agent validation logic not found"
        return 1
    fi

    # Validate test type validation exists
    log_check "Test: Test type validation logic exists"
    if grep -q "crashloopbackoff" "$main_script"; then
        log_success "Test type validation logic found"
    else
        log_error "Test type validation logic not found"
        return 1
    fi

    # Validate flag parsing exists
    log_check "Test: Flag parsing logic exists"
    if grep -q -- "--debug" "$main_script" && grep -q -- "--json" "$main_script"; then
        log_success "Flag parsing logic found"
    else
        log_error "Flag parsing logic not found"
        return 1
    fi

    return 0
}

# Test config generation logic
test_config_generation() {
    log_info "Testing config generation..."

    local config_gen="${LIB_DIR}/config-generator.sh"

    # Validate syntax
    if ! validate_syntax "$config_gen"; then
        return 1
    fi

    # Check for required functions
    log_check "Test: Required functions exist"
    local required_funcs=("usage" "validate_agent" "validate_prerequisites" "generate_config")
    local missing=0

    for func in "${required_funcs[@]}"; do
        if grep -q "^${func}()" "$config_gen" || grep -q "^${func} ()" "$config_gen"; then
            log_success "Function found: $func"
        else
            log_error "Function not found: $func"
            ((missing++))
        fi
    done

    if [ $missing -gt 0 ]; then
        return 1
    fi

    # Check template references
    log_check "Test: Template directory reference exists"
    if grep -q "config-templates" "$config_gen"; then
        log_success "Template directory reference found"
    else
        log_error "Template directory reference not found"
        return 1
    fi

    # Check debug mode handling
    log_check "Test: Debug mode handling exists"
    if grep -q "agent_debug" "$config_gen"; then
        log_success "Debug mode handling found"
    else
        log_error "Debug mode handling not found"
        return 1
    fi

    return 0
}

# Test log monitoring functions
test_log_monitoring() {
    log_info "Testing log monitoring..."

    local log_monitor="${LIB_DIR}/log-monitor.sh"

    # Validate syntax
    if ! validate_syntax "$log_monitor"; then
        return 1
    fi

    # Check for wait_for_pattern function
    log_check "Test: wait_for_pattern function exists"
    if grep -q "wait_for_pattern()" "$log_monitor"; then
        log_success "wait_for_pattern function found"
    else
        log_error "wait_for_pattern function not found"
        return 1
    fi

    # Check timeout handling
    log_check "Test: Timeout handling exists"
    if grep -q "timeout" "$log_monitor"; then
        log_success "Timeout handling found"
    else
        log_error "Timeout handling not found"
        return 1
    fi

    # Check pattern matching logic
    log_check "Test: Pattern matching logic exists"
    if grep -q "grep" "$log_monitor"; then
        log_success "Pattern matching logic found"
    else
        log_error "Pattern matching logic not found"
        return 1
    fi

    return 0
}

# Test report generation
test_report_generation() {
    log_info "Testing report generation..."

    local report_gen="${LIB_DIR}/report-generator.sh"

    # Validate syntax
    if ! validate_syntax "$report_gen"; then
        return 1
    fi

    # Check for generate_report function
    log_check "Test: generate_report function exists"
    if grep -q "generate_report()" "$report_gen"; then
        log_success "generate_report function found"
    else
        log_error "generate_report function not found"
        return 1
    fi

    # Check output format handling
    log_check "Test: Output format handling exists"
    if grep -q "json\|human" "$report_gen"; then
        log_success "Output format handling found"
    else
        log_error "Output format handling not found"
        return 1
    fi

    # Check metadata parsing
    log_check "Test: Metadata parsing exists"
    if grep -q "test.meta" "$report_gen"; then
        log_success "Metadata parsing found"
    else
        log_error "Metadata parsing not found"
        return 1
    fi

    return 0
}

# Test failure induction scripts
test_failure_induction() {
    log_info "Testing failure induction scripts..."

    local crashloop_script="${FAILURE_DIR}/01_induce_failure_crashloopbackoff.sh"

    # Validate syntax
    if ! validate_syntax "$crashloop_script"; then
        return 1
    fi

    # Check for start/stop commands
    log_check "Test: start/stop command handling exists"
    if grep -q "start)" "$crashloop_script" && grep -q "stop)" "$crashloop_script"; then
        log_success "start/stop command handling found"
    else
        log_error "start/stop command handling not found"
        return 1
    fi

    # Check TIMEOUT output
    log_check "Test: TIMEOUT output exists"
    if grep -q "TIMEOUT=" "$crashloop_script"; then
        log_success "TIMEOUT output found"
    else
        log_error "TIMEOUT output not found"
        return 1
    fi

    # Check namespace handling
    log_check "Test: Namespace handling exists"
    if grep -q "TEST_NAMESPACE\|NAMESPACE" "$crashloop_script"; then
        log_success "Namespace handling found"
    else
        log_warning "Namespace handling not found"
    fi

    # Check kubectl usage
    log_check "Test: kubectl commands exist"
    if grep -q "kubectl" "$crashloop_script"; then
        log_success "kubectl commands found"
    else
        log_error "kubectl commands not found"
        return 1
    fi

    return 0
}

# Test error handling in main script
test_error_handling() {
    log_info "Testing error handling..."

    local main_script="${TESTS_DIR}/run-live-test.sh"

    # Check for cleanup function
    log_check "Test: cleanup() function exists"
    if grep -q "^cleanup()" "$main_script"; then
        log_success "cleanup() function found"
    else
        log_error "cleanup() function not found"
        return 1
    fi

    # Check trap setup
    log_check "Test: trap setup exists"
    if grep -q "trap cleanup" "$main_script"; then
        log_success "trap setup found"
    else
        log_error "trap setup not found"
        return 1
    fi

    # Check set -e (exit on error)
    log_check "Test: set -euo pipefail exists"
    if grep -q "set -euo pipefail" "$main_script"; then
        log_success "set -euo pipefail found"
    else
        log_warning "set -euo pipefail not found"
    fi

    # Check prerequisite validation
    log_check "Test: validate_prerequisites function exists"
    if grep -q "validate_prerequisites()" "$main_script"; then
        log_success "validate_prerequisites function found"
    else
        log_error "validate_prerequisites function not found"
        return 1
    fi

    return 0
}

# Test integration points
test_integration_points() {
    log_info "Testing integration points..."

    local main_script="${TESTS_DIR}/run-live-test.sh"

    # Check config-generator invocation
    log_check "Test: config-generator.sh invocation exists"
    if grep -q "config-generator.sh" "$main_script"; then
        log_success "config-generator.sh invocation found"
    else
        log_error "config-generator.sh invocation not found"
        return 1
    fi

    # Check log-monitor sourcing
    log_check "Test: log-monitor.sh sourcing exists"
    if grep -q "source.*log-monitor.sh" "$main_script"; then
        log_success "log-monitor.sh sourcing found"
    else
        log_error "log-monitor.sh sourcing not found"
        return 1
    fi

    # Check report-generator sourcing
    log_check "Test: report-generator.sh sourcing exists"
    if grep -q "source.*report-generator.sh" "$main_script"; then
        log_success "report-generator.sh sourcing found"
    else
        log_error "report-generator.sh sourcing not found"
        return 1
    fi

    # Check failure script invocation
    log_check "Test: Failure script invocation exists"
    if grep -q "induce_failure" "$main_script"; then
        log_success "Failure script invocation found"
    else
        log_error "Failure script invocation not found"
        return 1
    fi

    # Check nightcrier binary invocation
    log_check "Test: Nightcrier binary invocation exists"
    if grep -q "NIGHTCRIER_BIN\|nightcrier.*--config" "$main_script"; then
        log_success "Nightcrier binary invocation found"
    else
        log_error "Nightcrier binary invocation not found"
        return 1
    fi

    return 0
}

# Validate directory structure
validate_directory_structure() {
    log_info "Validating directory structure..."

    local required_dirs=(
        "$TESTS_DIR"
        "$LIB_DIR"
        "$FAILURE_DIR"
        "${TESTS_DIR}/config-templates"
    )

    for dir in "${required_dirs[@]}"; do
        log_check "Directory exists: $(basename "$dir")"
        if [ -d "$dir" ]; then
            log_success "Directory exists: $dir"
        else
            log_error "Directory missing: $dir"
        fi
    done
}

# Validate required files exist
validate_required_files() {
    log_info "Validating required files..."

    local required_files=(
        "${TESTS_DIR}/run-live-test.sh"
        "${LIB_DIR}/config-generator.sh"
        "${LIB_DIR}/log-monitor.sh"
        "${LIB_DIR}/report-generator.sh"
        "${FAILURE_DIR}/01_induce_failure_crashloopbackoff.sh"
        "${TESTS_DIR}/README.md"
    )

    for file in "${required_files[@]}"; do
        log_check "File exists: $(basename "$file")"
        if [ -f "$file" ]; then
            log_success "File exists: $file"
        else
            log_error "File missing: $file"
        fi
    done
}

# Validate config templates
validate_config_templates() {
    log_info "Validating config templates..."

    local template_dir="${TESTS_DIR}/config-templates"
    local agents=("claude" "codex" "gemini")

    for agent in "${agents[@]}"; do
        local template="${template_dir}/test-${agent}.yaml.tmpl"
        log_check "Template exists: test-${agent}.yaml.tmpl"
        if [ -f "$template" ]; then
            log_success "Template exists: $template"

            # Check for required placeholders
            log_check "Template has required placeholders: $agent"
            if grep -q "\${ANTHROPIC_API_KEY}\|\${OPENAI_API_KEY}\|\${GEMINI_API_KEY}\|\${MCP_ENDPOINT}" "$template"; then
                log_success "Required placeholders found in $agent template"
            else
                log_warning "Missing placeholders in $agent template"
            fi
        else
            log_error "Template missing: $template"
        fi
    done
}

# Run shellcheck if available
run_shellcheck() {
    if ! command -v shellcheck &>/dev/null; then
        log_warning "shellcheck not available, skipping lint checks"
        return 0
    fi

    log_info "Running shellcheck on scripts..."

    local scripts=(
        "${TESTS_DIR}/run-live-test.sh"
        "${LIB_DIR}/config-generator.sh"
        "${LIB_DIR}/log-monitor.sh"
        "${LIB_DIR}/report-generator.sh"
        "${FAILURE_DIR}/01_induce_failure_crashloopbackoff.sh"
    )

    for script in "${scripts[@]}"; do
        log_check "shellcheck: $(basename "$script")"
        if shellcheck -x "$script" 2>&1; then
            log_success "shellcheck passed: $(basename "$script")"
        else
            log_warning "shellcheck found issues: $(basename "$script")"
        fi
    done
}

# Print summary
print_summary() {
    echo ""
    echo "=========================================="
    echo "Validation Summary"
    echo "=========================================="
    echo "Total checks: $CHECKS"
    echo -e "Passed:       ${GREEN}$((CHECKS - ERRORS - WARNINGS))${NC}"
    echo -e "Warnings:     ${YELLOW}${WARNINGS}${NC}"
    echo -e "Errors:       ${RED}${ERRORS}${NC}"
    echo "=========================================="

    if [ $ERRORS -eq 0 ]; then
        echo -e "${GREEN}Overall: PASS${NC}"
        echo ""
        echo "The test harness validation completed successfully."
        echo "All critical components are properly structured and integrated."
        return 0
    else
        echo -e "${RED}Overall: FAIL${NC}"
        echo ""
        echo "The test harness validation found ${ERRORS} error(s)."
        echo "Please review and fix the issues above."
        return 1
    fi
}

# Main validation flow
main() {
    # Parse arguments
    while [ $# -gt 0 ]; do
        case "$1" in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            *)
                echo "Unknown argument: $1"
                echo "Usage: $0 [--verbose]"
                exit 1
                ;;
        esac
    done

    echo "=========================================="
    echo "Test Harness Validation"
    echo "=========================================="
    echo "Repository: $REPO_ROOT"
    echo "Tests Dir:  $TESTS_DIR"
    echo "Verbose:    $VERBOSE"
    echo "=========================================="
    echo ""

    # Run all validation steps
    validate_directory_structure
    validate_required_files
    validate_config_templates

    # Validate syntax for all scripts
    log_info "Validating script syntax..."
    validate_syntax "${TESTS_DIR}/run-live-test.sh"
    validate_syntax "${LIB_DIR}/config-generator.sh"
    validate_syntax "${LIB_DIR}/log-monitor.sh"
    validate_syntax "${LIB_DIR}/report-generator.sh"
    validate_syntax "${FAILURE_DIR}/01_induce_failure_crashloopbackoff.sh"

    # Check executables
    log_info "Checking executable permissions..."
    check_executable "${TESTS_DIR}/run-live-test.sh"
    check_executable "${LIB_DIR}/config-generator.sh"
    check_executable "${FAILURE_DIR}/01_induce_failure_crashloopbackoff.sh"

    # Validate shebangs
    log_info "Validating shebangs..."
    validate_shebang "${TESTS_DIR}/run-live-test.sh"
    validate_shebang "${LIB_DIR}/config-generator.sh"
    validate_shebang "${LIB_DIR}/log-monitor.sh"
    validate_shebang "${LIB_DIR}/report-generator.sh"
    validate_shebang "${FAILURE_DIR}/01_induce_failure_crashloopbackoff.sh"

    # Run functional tests
    echo ""
    test_argument_parsing
    test_config_generation
    test_log_monitoring
    test_report_generation
    test_failure_induction
    test_error_handling
    test_integration_points

    # Run shellcheck
    echo ""
    run_shellcheck

    # Print summary
    echo ""
    print_summary
}

main "$@"
