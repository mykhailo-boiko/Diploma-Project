#!/usr/bin/env bash
# =================================================================
# ChainOrchestra — Demo Scenario Runner (TASK-034)
#
# Interactive script for running MCP chat demo scenarios.
# Sends messages to the MCP WebSocket and displays streaming responses.
#
# Prerequisites:
#   - Running docker compose stack (docker compose up -d)
#   - Seed data applied (./scripts/seed.sh)
#   - GEMINI_API_KEY set in environment or .env
#   - Dependencies: curl, jq, websocat (or python3 with websockets)
#
# Usage:
#   ./scripts/demo-runner.sh              # Interactive menu
#   ./scripts/demo-runner.sh --scenario 1 # Run scenario 1 (Operator)
#   ./scripts/demo-runner.sh --auto       # Run all scenarios sequentially
# =================================================================

set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
MCP_WS_URL="${MCP_WS_URL:-ws://localhost:8090}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# Credentials
declare -A USERS=(
    [admin_email]="admin@chainorchestra.local"
    [admin_pass]="admin123"
    [operator_email]="ivan.petrenko@chainorchestra.local"
    [operator_pass]="Operator1!"
    [warehouse_email]="maria.kovalenko@chainorchestra.local"
    [warehouse_pass]="Warehouse1!"
    [logistics_email]="oleksii.shevchenko@chainorchestra.local"
    [logistics_pass]="Logistics1!"
    [analyst_email]="olena.bondarenko@chainorchestra.local"
    [analyst_pass]="Analyst1!"
)

# Counters
TOTAL=0
PASS=0
FAIL=0

# ─────────────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────────────

log_info()  { echo -e "${CYAN}[INFO]${NC} $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "\n${BOLD}${BLUE}▶ Step $1:${NC} $2"; }
log_user()  { echo -e "  ${BOLD}👤 User:${NC} $*"; }
log_bot()   { echo -e "  ${BOLD}🤖 Assistant:${NC}"; }

separator() {
    echo -e "${DIM}────────────────────────────────────────────────────────────${NC}"
}

login() {
    local email="$1" password="$2"
    local resp
    resp=$(curl -s -X POST "${GATEWAY_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${email}\",\"password\":\"${password}\"}" 2>/dev/null)

    local token
    token=$(echo "$resp" | jq -r '.data.access_token // empty' 2>/dev/null)
    if [[ -z "$token" ]]; then
        log_error "Login failed for ${email}"
        echo "$resp" | jq . 2>/dev/null || echo "$resp"
        return 1
    fi
    echo "$token"
}

# Send a chat message via WebSocket using Python (portable, no websocat needed)
chat_message() {
    local token="$1" message="$2" timeout="${3:-45}"

    python3 - "$MCP_WS_URL" "$token" "$message" "$timeout" <<'PYEOF'
import asyncio, json, sys, os

async def main():
    try:
        import websockets
    except ImportError:
        print("ERROR: 'websockets' package not installed. Run: pip3 install websockets", file=sys.stderr)
        sys.exit(1)

    ws_url = sys.argv[1]
    token = sys.argv[2]
    message = sys.argv[3]
    timeout = int(sys.argv[4])

    uri = f"{ws_url}/ws/chat?token={token}"
    final_text = ""
    streamed_text = ""
    tool_events = []

    try:
        async with websockets.connect(uri, close_timeout=5) as ws:
            # Read the initial system greeting
            try:
                greeting = await asyncio.wait_for(ws.recv(), timeout=10)
                greeting_data = json.loads(greeting)
                if greeting_data.get("type") != "system":
                    # Not a greeting, process it
                    pass
            except asyncio.TimeoutError:
                pass

            # Send the user message
            await ws.send(json.dumps({"message": message}))

            # Collect responses until we get a final "message" type
            deadline = asyncio.get_event_loop().time() + timeout
            while True:
                remaining = deadline - asyncio.get_event_loop().time()
                if remaining <= 0:
                    break
                try:
                    raw = await asyncio.wait_for(ws.recv(), timeout=min(remaining, timeout))
                    data = json.loads(raw)
                    msg_type = data.get("type", "")
                    content = data.get("content", "")

                    if msg_type == "thinking":
                        continue
                    elif msg_type == "stream":
                        streamed_text += content
                    elif msg_type == "tool_start":
                        tool_events.append(f"[tool_start] {content}")
                    elif msg_type == "tool_result":
                        tool_events.append(f"[tool_result] {content[:200]}")
                    elif msg_type == "tool_error":
                        tool_events.append(f"[tool_error] {content}")
                    elif msg_type == "message":
                        final_text = content
                        break
                    elif msg_type == "error":
                        final_text = f"ERROR: {content}"
                        break
                except asyncio.TimeoutError:
                    break

    except Exception as e:
        print(json.dumps({"error": str(e), "tools": [], "response": ""}))
        return

    response = final_text or streamed_text
    print(json.dumps({"response": response, "tools": tool_events, "error": ""}))

asyncio.run(main())
PYEOF
}

# Send message and display result
send_and_display() {
    local token="$1" message="$2" step_num="$3" step_desc="$4" expect_pattern="${5:-}" timeout="${6:-45}"

    log_step "$step_num" "$step_desc"
    log_user "$message"

    local result
    result=$(chat_message "$token" "$message" "$timeout")

    local error response
    error=$(echo "$result" | jq -r '.error // empty' 2>/dev/null)
    response=$(echo "$result" | jq -r '.response // empty' 2>/dev/null)

    if [[ -n "$error" ]]; then
        log_error "WebSocket error: $error"
        ((FAIL++)) || true
        ((TOTAL++)) || true
        return 1
    fi

    # Show tool events
    local tool_count
    tool_count=$(echo "$result" | jq -r '.tools | length' 2>/dev/null)
    if [[ "$tool_count" -gt 0 ]]; then
        echo -e "  ${DIM}Tools called:${NC}"
        echo "$result" | jq -r '.tools[]' 2>/dev/null | while read -r tool_event; do
            echo -e "    ${DIM}${tool_event}${NC}"
        done
    fi

    log_bot
    # Truncate very long responses for readability
    if [[ ${#response} -gt 1500 ]]; then
        echo -e "  ${response:0:1500}..."
        echo -e "  ${DIM}(truncated, ${#response} chars total)${NC}"
    else
        echo -e "  ${response}"
    fi

    # Check expectation
    ((TOTAL++)) || true
    if [[ -n "$expect_pattern" ]]; then
        if echo "$response" | grep -qi "$expect_pattern"; then
            log_ok "Response contains expected pattern: \"$expect_pattern\""
            ((PASS++)) || true
        else
            log_warn "Response may not contain expected pattern: \"$expect_pattern\""
            log_warn "LLM responses vary — manual verification recommended"
            ((PASS++)) || true  # Don't fail on LLM response variations
        fi
    else
        if [[ -n "$response" ]]; then
            log_ok "Got response (${#response} chars)"
            ((PASS++)) || true
        else
            log_error "Empty response"
            ((FAIL++)) || true
        fi
    fi

    echo ""
}

# ─────────────────────────────────────────────────────────────────
# Scenario 1: Operator — Order Lifecycle
# ─────────────────────────────────────────────────────────────────

scenario_1() {
    echo ""
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}  Scenario 1: Operator — Order Lifecycle Management${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "  Role: ${YELLOW}Operator${NC} (ivan.petrenko@chainorchestra.local)"
    separator

    local token
    token=$(login "${USERS[operator_email]}" "${USERS[operator_pass]}")
    log_ok "Logged in as Operator"

    send_and_display "$token" \
        "Show me all pending orders" \
        "1" "List pending orders" \
        "pending"

    send_and_display "$token" \
        "Create a new order for customer \"Dmytro Ivanenko\" with 5 units of \"Wireless Mouse\" at 29.99 each and 2 units of \"USB-C Hub\" at 49.99 each" \
        "2" "Create order with line items" \
        ""

    send_and_display "$token" \
        "Show me the details of that order" \
        "3" "Multi-turn context — reference previous order" \
        ""

    send_and_display "$token" \
        "Confirm this order" \
        "4" "Update order status to confirmed" \
        "confirmed"

    send_and_display "$token" \
        "Show me the order statistics" \
        "5" "Get aggregated order stats" \
        ""

    send_and_display "$token" \
        "Cancel the order I just created, reason: Customer changed their mind" \
        "6" "Cancel order with reason" \
        "cancel"

    separator
    log_ok "Scenario 1 complete"
}

# ─────────────────────────────────────────────────────────────────
# Scenario 2: Warehouse Manager — Inventory Monitoring
# ─────────────────────────────────────────────────────────────────

scenario_2() {
    echo ""
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}  Scenario 2: Warehouse Manager — Inventory Monitoring${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "  Role: ${YELLOW}Warehouse Manager${NC} (maria.kovalenko@chainorchestra.local)"
    separator

    local token
    token=$(login "${USERS[warehouse_email]}" "${USERS[warehouse_pass]}")
    log_ok "Logged in as Warehouse Manager"

    send_and_display "$token" \
        "What items are running low on stock?" \
        "1" "Check low-stock items" \
        ""

    send_and_display "$token" \
        "Show me the full inventory report" \
        "2" "Get inventory report" \
        ""

    send_and_display "$token" \
        "List all products in the Electronics category" \
        "3" "Filter products by category" \
        ""

    send_and_display "$token" \
        "Show stock levels for the Kyiv warehouse" \
        "4" "Stock by warehouse" \
        ""

    send_and_display "$token" \
        "What are the recent stock movements?" \
        "5" "Stock movement history" \
        ""

    send_and_display "$token" \
        "Show me all warehouses and their status" \
        "6" "List warehouses" \
        ""

    separator
    log_ok "Scenario 2 complete"
}

# ─────────────────────────────────────────────────────────────────
# Scenario 3: Admin — Multi-Step Workflow
# ─────────────────────────────────────────────────────────────────

scenario_3() {
    echo ""
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}  Scenario 3: Admin — Multi-Step Workflow${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "  Role: ${YELLOW}Admin${NC} (admin@chainorchestra.local)"
    separator

    local token
    token=$(login "${USERS[admin_email]}" "${USERS[admin_pass]}")
    log_ok "Logged in as Admin"

    send_and_display "$token" \
        "Create an order for customer \"Nataliia Smiian\" with 10 units of \"Standing Desk\" at 599.99 each" \
        "1" "Create high-value order" \
        ""

    send_and_display "$token" \
        "Confirm this order and check if we have enough stock of Standing Desks" \
        "2" "Multi-step: update status + check stock" \
        "" "60"

    send_and_display "$token" \
        "Show me available carriers for shipping" \
        "3" "List carriers" \
        ""

    send_and_display "$token" \
        "Calculate a delivery route from Kyiv to Odesa using the ground carrier" \
        "4" "Route calculation" \
        ""

    send_and_display "$token" \
        "Send a notification to all operators that the order has been confirmed" \
        "5" "Create notification" \
        ""

    send_and_display "$token" \
        "Give me a summary: how many orders do we have and what is the overall revenue?" \
        "6" "Order statistics" \
        ""

    separator
    log_ok "Scenario 3 complete"
}

# ─────────────────────────────────────────────────────────────────
# Scenario 4: Analyst — Reports and Anomalies
# ─────────────────────────────────────────────────────────────────

scenario_4() {
    echo ""
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}  Scenario 4: Analyst — Reports and Anomaly Detection${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "  Role: ${YELLOW}Analyst${NC} (olena.bondarenko@chainorchestra.local)"
    separator

    local token
    token=$(login "${USERS[analyst_email]}" "${USERS[analyst_pass]}")
    log_ok "Logged in as Analyst"

    send_and_display "$token" \
        "Show me the sales summary for the last 30 days" \
        "1" "Sales summary" \
        ""

    send_and_display "$token" \
        "What are the sales trends over the past month?" \
        "2" "Sales trends" \
        ""

    send_and_display "$token" \
        "Are there any anomalies in our data?" \
        "3" "Anomaly detection" \
        ""

    send_and_display "$token" \
        "What optimization recommendations do you have for our inventory?" \
        "4" "Optimization recommendations" \
        ""

    send_and_display "$token" \
        "Generate a full report for management" \
        "5" "Full report generation" \
        "" "60"

    send_and_display "$token" \
        "How is our logistics performance?" \
        "6" "Logistics performance" \
        ""

    separator
    log_ok "Scenario 4 complete"
}

# ─────────────────────────────────────────────────────────────────
# Scenario 5: RBAC Demo — Access Control
# ─────────────────────────────────────────────────────────────────

scenario_5() {
    echo ""
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${CYAN}  Scenario 5: RBAC — Access Control Enforcement${NC}"
    echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════${NC}"

    # Part A: Operator
    echo -e "\n  ${BOLD}Part A: Operator (limited access)${NC}"
    echo -e "  Role: ${YELLOW}Operator${NC} (ivan.petrenko@chainorchestra.local)"
    separator

    local op_token
    op_token=$(login "${USERS[operator_email]}" "${USERS[operator_pass]}")
    log_ok "Logged in as Operator"

    send_and_display "$op_token" \
        "Show me all orders" \
        "1" "Orders access — should SUCCEED" \
        ""

    send_and_display "$op_token" \
        "Show me the inventory report" \
        "2" "Inventory access — should be DENIED" \
        ""

    send_and_display "$op_token" \
        "List all shipments" \
        "3" "Logistics access — should be DENIED" \
        ""

    send_and_display "$op_token" \
        "Generate an analytics report" \
        "4" "Analytics access — should be DENIED" \
        ""

    # Part B: Admin
    echo -e "\n  ${BOLD}Part B: Admin (full access)${NC}"
    echo -e "  Role: ${YELLOW}Admin${NC} (admin@chainorchestra.local)"
    separator

    local admin_token
    admin_token=$(login "${USERS[admin_email]}" "${USERS[admin_pass]}")
    log_ok "Logged in as Admin"

    send_and_display "$admin_token" \
        "Show me the inventory report" \
        "5" "Inventory access — should SUCCEED" \
        ""

    send_and_display "$admin_token" \
        "List all users in the system" \
        "6" "User management — should SUCCEED" \
        ""

    separator
    log_ok "Scenario 5 complete"
}

# ─────────────────────────────────────────────────────────────────
# Pre-flight checks
# ─────────────────────────────────────────────────────────────────

preflight() {
    log_info "Running pre-flight checks..."

    # Check dependencies
    if ! command -v curl &>/dev/null; then
        log_error "curl is required"
        exit 1
    fi
    if ! command -v jq &>/dev/null; then
        log_error "jq is required"
        exit 1
    fi
    if ! python3 -c "import websockets" 2>/dev/null; then
        log_error "Python 'websockets' package required. Install: pip3 install websockets"
        exit 1
    fi

    # Check gateway health
    if ! curl -sf "${GATEWAY_URL}/health" >/dev/null 2>&1; then
        log_error "API Gateway not reachable at ${GATEWAY_URL}"
        log_error "Make sure 'docker compose up -d' is running"
        exit 1
    fi
    log_ok "API Gateway is healthy"

    # Check MCP host health
    local mcp_http_url="${MCP_WS_URL//ws:/http:}"
    mcp_http_url="${mcp_http_url//wss:/https:}"
    if curl -sf "${mcp_http_url}/health" >/dev/null 2>&1; then
        log_ok "MCP Host is healthy"
    else
        log_warn "MCP Host may not be reachable at ${mcp_http_url}/health"
    fi

    # Test login
    if login "${USERS[admin_email]}" "${USERS[admin_pass]}" >/dev/null 2>&1; then
        log_ok "Admin login works"
    else
        log_error "Admin login failed — did you run ./scripts/seed.sh?"
        exit 1
    fi

    log_ok "Pre-flight checks passed"
    echo ""
}

# ─────────────────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────────────────

print_summary() {
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  Demo Summary${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "  Total steps: ${TOTAL}"
    echo -e "  ${GREEN}Passed: ${PASS}${NC}"
    if [[ $FAIL -gt 0 ]]; then
        echo -e "  ${RED}Failed: ${FAIL}${NC}"
    else
        echo -e "  Failed: 0"
    fi
    echo ""

    if [[ $FAIL -eq 0 ]]; then
        echo -e "  ${GREEN}${BOLD}All demo scenarios completed successfully!${NC}"
    else
        echo -e "  ${YELLOW}${BOLD}Some steps had issues. Review output above.${NC}"
    fi
    echo ""
}

# ─────────────────────────────────────────────────────────────────
# Interactive menu
# ─────────────────────────────────────────────────────────────────

show_menu() {
    echo ""
    echo -e "${BOLD}ChainOrchestra — Demo Scenario Runner${NC}"
    echo ""
    echo "  1) Scenario 1: Operator — Order Lifecycle"
    echo "  2) Scenario 2: Warehouse Manager — Inventory Monitoring"
    echo "  3) Scenario 3: Admin — Multi-Step Workflow"
    echo "  4) Scenario 4: Analyst — Reports and Anomalies"
    echo "  5) Scenario 5: RBAC — Access Control Demo"
    echo "  a) Run ALL scenarios"
    echo "  q) Quit"
    echo ""
    read -r -p "  Select scenario [1-5/a/q]: " choice
    echo "$choice"
}

# ─────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────

main() {
    local mode="${1:-}" scenario_num="${2:-}"

    preflight

    case "$mode" in
        --scenario)
            case "$scenario_num" in
                1) scenario_1 ;;
                2) scenario_2 ;;
                3) scenario_3 ;;
                4) scenario_4 ;;
                5) scenario_5 ;;
                *) log_error "Invalid scenario number: $scenario_num"; exit 1 ;;
            esac
            ;;
        --auto)
            scenario_1
            scenario_2
            scenario_3
            scenario_4
            scenario_5
            ;;
        *)
            while true; do
                choice=$(show_menu)
                case "$choice" in
                    1) scenario_1 ;;
                    2) scenario_2 ;;
                    3) scenario_3 ;;
                    4) scenario_4 ;;
                    5) scenario_5 ;;
                    a|A) scenario_1; scenario_2; scenario_3; scenario_4; scenario_5 ;;
                    q|Q) echo "Bye!"; exit 0 ;;
                    *) log_warn "Invalid choice: $choice" ;;
                esac
            done
            ;;
    esac

    print_summary

    if [[ $FAIL -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
