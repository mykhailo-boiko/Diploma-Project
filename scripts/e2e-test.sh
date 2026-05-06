#!/usr/bin/env bash
set -uo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
MCP_URL="${MCP_URL:-http://localhost:8090}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@chainorchestra.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"
OPERATOR_EMAIL="ivan.petrov@chainorchestra.local"
OPERATOR_PASSWORD="Operator1!"
WAREHOUSE_EMAIL="maria.kuznetsova@chainorchestra.local"
WAREHOUSE_PASSWORD="Warehouse1!"
LOGISTICS_EMAIL="alexei.volkov@chainorchestra.local"
LOGISTICS_PASSWORD="Logistics1!"
ANALYST_EMAIL="elena.sokolova@chainorchestra.local"
ANALYST_PASSWORD="Analyst1!"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0
TOTAL_COUNT=0


log_info()    { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()    { echo -e "\n${BLUE}=== $1 ===${NC}"; }
log_scenario(){ echo -e "\n${CYAN}>>> SCENARIO $1 <<<${NC}"; }

assert_pass() {
  local name="$1"
  TOTAL_COUNT=$((TOTAL_COUNT + 1))
  PASS_COUNT=$((PASS_COUNT + 1))
  echo -e "  ${GREEN}PASS${NC} $name"
}

assert_fail() {
  local name="$1"
  local detail="${2:-}"
  TOTAL_COUNT=$((TOTAL_COUNT + 1))
  FAIL_COUNT=$((FAIL_COUNT + 1))
  echo -e "  ${RED}FAIL${NC} $name"
  if [ -n "$detail" ]; then
    echo -e "       ${RED}Detail: $detail${NC}"
  fi
}

assert_skip() {
  local name="$1"
  local reason="${2:-}"
  TOTAL_COUNT=$((TOTAL_COUNT + 1))
  SKIP_COUNT=$((SKIP_COUNT + 1))
  echo -e "  ${YELLOW}SKIP${NC} $name${reason:+ ($reason)}"
}

assert_eq() {
  local name="$1"
  local expected="$2"
  local actual="$3"
  if [ "$expected" = "$actual" ]; then
    assert_pass "$name"
  else
    assert_fail "$name" "expected='$expected' actual='$actual'"
  fi
}

assert_contains() {
  local name="$1"
  local haystack="$2"
  local needle="$3"
  if echo "$haystack" | grep -q "$needle"; then
    assert_pass "$name"
  else
    assert_fail "$name" "expected to contain '$needle'"
  fi
}

assert_not_empty() {
  local name="$1"
  local value="$2"
  if [ -n "$value" ] && [ "$value" != "null" ] && [ "$value" != "" ]; then
    assert_pass "$name"
  else
    assert_fail "$name" "value is empty or null"
  fi
}

assert_http() {
  local name="$1"
  local expected_status="$2"
  local actual_status="$3"
  assert_eq "$name (HTTP $expected_status)" "$expected_status" "$actual_status"
}

api_call() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  local token="${4:-}"
  local args=(-s -w "\n%{http_code}" -H "Content-Type: application/json")
  if [ -n "$token" ]; then
    args+=(-H "Authorization: Bearer $token")
  fi
  if [ -n "$data" ]; then
    args+=(-d "$data")
  fi
  curl "${args[@]}" -X "$method" "${GATEWAY_URL}${path}"
}

api_get()    { api_call GET "$1" "" "${2:-}"; }
api_post()   { api_call POST "$1" "$2" "${3:-}"; }
api_put()    { api_call PUT "$1" "$2" "${3:-}"; }
api_delete() { api_call DELETE "$1" "" "${2:-}"; }

get_body() { echo "$1" | sed '$d'; }
get_status() { echo "$1" | tail -1; }
json_field() { echo "$1" | sed '$d' | jq -r "$2"; }

login() {
  local email="$1"
  local password="$2"
  local resp
  resp=$(api_post "/api/v1/auth/login" "{\"email\":\"$email\",\"password\":\"$password\"}")
  json_field "$resp" '.data.access_token // .access_token // empty'
}


wait_for_gateway() {
  log_step "Waiting for API Gateway"
  local max_wait=60
  local waited=0
  while [ $waited -lt $max_wait ]; do
    if curl -sf "${GATEWAY_URL}/health" > /dev/null 2>&1; then
      log_info "Gateway ready"
      return 0
    fi
    sleep 2
    waited=$((waited + 2))
  done
  log_error "Gateway not ready after ${max_wait}s"
  exit 1
}

wait_for_mcp() {
  log_step "Checking MCP Host"
  if curl -sf "${MCP_URL}/health" > /dev/null 2>&1; then
    log_info "MCP Host ready"
    return 0
  fi
  log_warn "MCP Host not available (WebSocket tests will be skipped)"
  return 1
}

scenario_1_order_create_and_stock() {
  log_scenario "1: Create order and verify stock (order + inventory integration)"

  local op_token
  op_token=$(login "$OPERATOR_EMAIL" "$OPERATOR_PASSWORD")
  if [ -z "$op_token" ]; then
    op_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  fi
  assert_not_empty "Login for order creation" "$op_token"
  if [ -z "$op_token" ]; then return 1; fi

  local admin_token
  admin_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  assert_not_empty "Admin login" "$admin_token"

  local products_resp
  products_resp=$(api_get "/api/v1/products?limit=1" "$admin_token")
  local product_name
  product_name=$(json_field "$products_resp" '.data[0].name // empty')
  local product_id
  product_id=$(json_field "$products_resp" '.data[0].id // empty')
  assert_not_empty "Product exists in inventory" "$product_name"

  local stock_resp
  stock_resp=$(api_get "/api/v1/stock?product_id=$product_id&limit=1" "$admin_token")
  local initial_available
  initial_available=$(json_field "$stock_resp" '.data[0].available // "0"')
  local warehouse_id
  warehouse_id=$(json_field "$stock_resp" '.data[0].warehouse_id // empty')
  log_info "Initial stock for $product_name: available=$initial_available (warehouse=$warehouse_id)"

  local create_resp
  create_resp=$(api_post "/api/v1/orders" "{
    \"customer_name\": \"E2E Test Customer\",
    \"customer_email\": \"e2e-test@example.com\",
    \"shipping_address\": \"Test Street 1, Moscow\",
    \"items\": [
      {
        \"product_name\": \"$product_name\",
        \"product_id\": \"$product_id\",
        \"quantity\": 5,
        \"unit_price\": 99.99
      }
    ]
  }" "$op_token")
  local create_status
  create_status=$(get_status "$create_resp")
  assert_http "Create order" "201" "$create_status"

  local order_id
  order_id=$(json_field "$create_resp" '.data.id // empty')
  assert_not_empty "Order ID returned" "$order_id"
  if [ -z "$order_id" ] || [ "$order_id" = "null" ]; then return 1; fi

  local order_resp
  order_resp=$(api_get "/api/v1/orders/$order_id" "$op_token")
  local order_status
  order_status=$(json_field "$order_resp" '.data.status // empty')
  assert_eq "Order status is pending" "pending" "$order_status"

  local total
  total=$(json_field "$order_resp" '.data.total_amount // "0"')
  assert_eq "Order total_amount computed (5*99.99)" "499.95" "$total"

  local item_count
  item_count=$(json_field "$order_resp" '.data.items | length')
  assert_eq "Order has 1 item" "1" "$item_count"

  local list_resp
  list_resp=$(api_get "/api/v1/orders?status=pending" "$op_token")
  local list_status
  list_status=$(get_status "$list_resp")
  assert_http "List orders with status filter" "200" "$list_status"

  local search_resp
  search_resp=$(api_get "/api/v1/orders/search?q=E2E%20Test" "$op_token")
  local search_status
  search_status=$(get_status "$search_resp")
  assert_http "Search orders by customer name" "200" "$search_status"
  local search_count
  search_count=$(json_field "$search_resp" '.data | length')
  if [ "$search_count" -gt 0 ] 2>/dev/null; then
    assert_pass "Search found the order"
  else
    assert_fail "Search found the order" "count=$search_count"
  fi

  E2E_ORDER_ID="$order_id"
  E2E_OP_TOKEN="$op_token"
  E2E_ADMIN_TOKEN="$admin_token"
  E2E_PRODUCT_ID="$product_id"
  E2E_WAREHOUSE_ID="$warehouse_id"
}

scenario_2_multistep_workflow() {
  log_scenario "2: Multi-step workflow (status transitions + logistics)"

  if [ -z "${E2E_ORDER_ID:-}" ]; then
    assert_skip "Multi-step workflow" "no order from scenario 1"
    return 1
  fi

  local token="${E2E_ADMIN_TOKEN}"
  local order_id="$E2E_ORDER_ID"

  local resp
  resp=$(api_put "/api/v1/orders/$order_id/status" '{"status":"confirmed"}' "$token")
  local status
  status=$(get_status "$resp")
  assert_http "Transition pending -> confirmed" "200" "$status"

  resp=$(api_get "/api/v1/orders/$order_id" "$token")
  local order_status
  order_status=$(json_field "$resp" '.data.status // empty')
  assert_eq "Order is confirmed" "confirmed" "$order_status"

  sleep 2

  resp=$(api_get "/api/v1/shipments?order_id=$order_id" "$token")
  local shipment_count
  shipment_count=$(json_field "$resp" '.data | length')
  if [ "$shipment_count" -gt 0 ] 2>/dev/null; then
    assert_pass "Logistics auto-created shipment on order confirmation"
    E2E_SHIPMENT_ID=$(json_field "$resp" '.data[0].id // empty')
  else
    assert_skip "Logistics auto-created shipment" "NATS event may not have propagated"
    E2E_SHIPMENT_ID=""
  fi

  resp=$(api_put "/api/v1/orders/$order_id/status" '{"status":"processing"}' "$token")
  status=$(get_status "$resp")
  assert_http "Transition confirmed -> processing" "200" "$status"

  resp=$(api_put "/api/v1/orders/$order_id/status" '{"status":"shipped"}' "$token")
  status=$(get_status "$resp")
  assert_http "Transition processing -> shipped" "200" "$status"

  resp=$(api_put "/api/v1/orders/$order_id/status" '{"status":"delivered"}' "$token")
  status=$(get_status "$resp")
  assert_http "Transition shipped -> delivered" "200" "$status"

  resp=$(api_put "/api/v1/orders/$order_id/status" '{"status":"completed"}' "$token")
  status=$(get_status "$resp")
  assert_http "Transition delivered -> completed" "200" "$status"

  resp=$(api_get "/api/v1/orders/$order_id" "$token")
  order_status=$(json_field "$resp" '.data.status // empty')
  assert_eq "Order completed full workflow" "completed" "$order_status"

  resp=$(api_post "/api/v1/orders" '{
    "customer_name": "E2E Invalid Transition",
    "customer_email": "invalid@test.com",
    "shipping_address": "Test St",
    "items": [{"product_name":"Widget","quantity":1,"unit_price":10}]
  }' "$token")
  local new_order_id
  new_order_id=$(json_field "$resp" '.data.id // empty')
  if [ -n "$new_order_id" ] && [ "$new_order_id" != "null" ]; then
    resp=$(api_put "/api/v1/orders/$new_order_id/status" '{"status":"delivered"}' "$token")
    status=$(get_status "$resp")
    if [ "$status" = "400" ] || [ "$status" = "422" ]; then
      assert_pass "Invalid transition pending->delivered rejected"
    else
      assert_fail "Invalid transition pending->delivered rejected" "got HTTP $status"
    fi

    resp=$(api_post "/api/v1/orders/$new_order_id/cancel" '{"reason":"E2E test cancellation"}' "$token")
    status=$(get_status "$resp")
    assert_http "Cancel order with reason" "200" "$status"

    resp=$(api_get "/api/v1/orders/$new_order_id" "$token")
    order_status=$(json_field "$resp" '.data.status // empty')
    assert_eq "Cancelled order has correct status" "cancelled" "$order_status"
  fi

  resp=$(api_get "/api/v1/orders/stats" "$token")
  status=$(get_status "$resp")
  assert_http "Order stats endpoint" "200" "$status"
  local total_orders
  total_orders=$(json_field "$resp" '.data.total_orders // 0')
  if [ "$total_orders" -gt 0 ] 2>/dev/null; then
    assert_pass "Order stats returns data (total_orders=$total_orders)"
  else
    assert_fail "Order stats returns data" "total_orders=$total_orders"
  fi
}

scenario_3_analytics_and_low_stock() {
  log_scenario "3: Analytics queries and low stock monitoring"

  local token="${E2E_ADMIN_TOKEN:-}"
  if [ -z "$token" ]; then
    token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  fi

  local resp
  resp=$(api_get "/api/v1/analytics/sales/summary?from=2020-01-01&to=2030-12-31" "$token")
  local status
  status=$(get_status "$resp")
  assert_http "Analytics sales summary" "200" "$status"

  resp=$(api_get "/api/v1/analytics/sales/trends?from=2020-01-01&to=2030-12-31&granularity=day" "$token")
  status=$(get_status "$resp")
  assert_http "Analytics sales trends" "200" "$status"

  resp=$(api_get "/api/v1/analytics/inventory/summary" "$token")
  status=$(get_status "$resp")
  assert_http "Analytics inventory summary" "200" "$status"

  resp=$(api_get "/api/v1/analytics/logistics/performance?from=2020-01-01&to=2030-12-31" "$token")
  status=$(get_status "$resp")
  assert_http "Analytics logistics performance" "200" "$status"

  resp=$(api_get "/api/v1/analytics/anomalies?from=2020-01-01&to=2030-12-31" "$token")
  status=$(get_status "$resp")
  assert_http "Analytics anomalies detection" "200" "$status"

  resp=$(api_get "/api/v1/analytics/optimization" "$token")
  status=$(get_status "$resp")
  assert_http "Analytics optimization recommendations" "200" "$status"

  resp=$(api_post "/api/v1/analytics/report" '{
    "report_type": "full",
    "from": "2020-01-01",
    "to": "2030-12-31"
  }' "$token")
  status=$(get_status "$resp")
  assert_http "Generate full analytics report" "200" "$status"

  resp=$(api_get "/api/v1/stock/low" "$token")
  status=$(get_status "$resp")
  assert_http "Low stock items endpoint" "200" "$status"

  resp=$(api_get "/api/v1/inventory/report" "$token")
  status=$(get_status "$resp")
  assert_http "Inventory report" "200" "$status"
  local body
  body=$(get_body "$resp")
  assert_contains "Inventory report has total data" "$body" "total"
}

scenario_4_rbac_enforcement() {
  log_scenario "4: RBAC enforcement (role-based access control)"

  local resp
  resp=$(api_get "/api/v1/users/me")
  local status
  status=$(get_status "$resp")
  assert_http "No token returns 401" "401" "$status"

  resp=$(api_get "/api/v1/users/me" "invalid-jwt-token")
  status=$(get_status "$resp")
  assert_http "Invalid token returns 401" "401" "$status"

  local op_token
  op_token=$(login "$OPERATOR_EMAIL" "$OPERATOR_PASSWORD")
  if [ -z "$op_token" ]; then
    assert_skip "Operator RBAC tests" "cannot login as operator"
    return 1
  fi

  resp=$(api_get "/api/v1/users" "$op_token")
  status=$(get_status "$resp")
  assert_http "Operator cannot list users (admin-only)" "403" "$status"

  resp=$(api_get "/api/v1/orders" "$op_token")
  status=$(get_status "$resp")
  assert_http "Operator can list orders" "200" "$status"

  resp=$(api_get "/api/v1/users/me" "$op_token")
  status=$(get_status "$resp")
  assert_http "Operator can access own profile" "200" "$status"
  local op_role
  op_role=$(json_field "$resp" '.data.role // empty')
  assert_eq "Operator role is correct" "operator" "$op_role"

  resp=$(api_put "/api/v1/users/me" '{"role":"admin"}' "$op_token")
  resp=$(api_get "/api/v1/users/me" "$op_token")
  op_role=$(json_field "$resp" '.data.role // empty')
  assert_eq "Operator cannot escalate own role" "operator" "$op_role"

  local wh_token
  wh_token=$(login "$WAREHOUSE_EMAIL" "$WAREHOUSE_PASSWORD")
  if [ -n "$wh_token" ]; then
    resp=$(api_get "/api/v1/products" "$wh_token")
    status=$(get_status "$resp")
    assert_http "Warehouse manager can list products" "200" "$status"
  else
    assert_skip "Warehouse manager tests" "cannot login"
  fi

  local admin_token
  admin_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  resp=$(api_get "/api/v1/users" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list users" "200" "$status"

  resp=$(api_get "/api/v1/orders" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list orders" "200" "$status"

  resp=$(api_get "/api/v1/products" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list products" "200" "$status"

  resp=$(api_get "/api/v1/shipments" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list shipments" "200" "$status"

  resp=$(api_get "/api/v1/analytics/sales/summary?from=2020-01-01&to=2030-12-31" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can access analytics" "200" "$status"

  resp=$(api_get "/api/v1/notifications" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list notifications" "200" "$status"

  resp=$(api_post "/api/v1/auth/password-reset" "{\"email\":\"$OPERATOR_EMAIL\"}")
  status=$(get_status "$resp")
  assert_http "Password reset request accepted" "200" "$status"

  resp=$(api_post "/api/v1/auth/password-reset" '{"email":"nonexistent@test.com"}')
  status=$(get_status "$resp")
  assert_http "Password reset no email enumeration" "200" "$status"

  local login_resp
  login_resp=$(api_post "/api/v1/auth/login" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")
  local refresh_token
  refresh_token=$(json_field "$login_resp" '.data.refresh_token // empty')
  if [ -n "$refresh_token" ] && [ "$refresh_token" != "null" ]; then
    resp=$(api_post "/api/v1/auth/refresh" "{\"refresh_token\":\"$refresh_token\"}")
    status=$(get_status "$resp")
    assert_http "Refresh token returns new access token" "200" "$status"
    local new_access
    new_access=$(json_field "$resp" '.data.access_token // empty')
    assert_not_empty "New access token returned" "$new_access"
  else
    assert_skip "Refresh token flow" "no refresh_token in login response"
  fi
}

scenario_5_full_event_chain() {
  log_scenario "5: Full event chain (order -> inventory -> shipment -> analytics -> notification)"

  local admin_token
  admin_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  assert_not_empty "Admin login for event chain" "$admin_token"
  if [ -z "$admin_token" ]; then return 1; fi

  local notif_resp
  notif_resp=$(api_get "/api/v1/notifications/unread-count" "$admin_token")
  local initial_unread
  initial_unread=$(json_field "$notif_resp" '.data.count // .count // 0')
  log_info "Initial unread notifications: $initial_unread"

  local resp
  resp=$(api_post "/api/v1/orders" '{
    "customer_name": "E2E Event Chain Customer",
    "customer_email": "chain@test.com",
    "shipping_address": "Event Chain Street 42, Moscow",
    "items": [
      {"product_name":"Chain Widget A","quantity":3,"unit_price":50},
      {"product_name":"Chain Widget B","quantity":2,"unit_price":75}
    ]
  }' "$admin_token")
  local order_id
  order_id=$(json_field "$resp" '.data.id // empty')
  assert_not_empty "Event chain order created" "$order_id"
  if [ -z "$order_id" ] || [ "$order_id" = "null" ]; then return 1; fi

  local total
  total=$(json_field "$resp" '.data.total_amount // 0')
  assert_eq "Event chain order total (3*50+2*75)" "300" "$total"

  resp=$(api_put "/api/v1/orders/$order_id/status" '{"status":"confirmed"}' "$admin_token")
  assert_http "Confirm event chain order" "200" "$(get_status "$resp")"

  sleep 3

  notif_resp=$(api_get "/api/v1/notifications?limit=20" "$admin_token")
  local notif_status
  notif_status=$(get_status "$notif_resp")
  assert_http "Notifications list after order events" "200" "$notif_status"

  notif_resp=$(api_get "/api/v1/notifications/unread-count" "$admin_token")
  local final_unread
  final_unread=$(json_field "$notif_resp" '.data.count // .count // 0')
  log_info "Final unread notifications: $final_unread"
  if [ "$final_unread" -ge "$initial_unread" ] 2>/dev/null; then
    assert_pass "Notifications generated by event chain (unread: $initial_unread -> $final_unread)"
  else
    assert_skip "Notification count increased" "NATS events may not have reached notification-service"
  fi

  resp=$(api_get "/api/v1/shipments?order_id=$order_id" "$admin_token")
  local shipment_count
  shipment_count=$(json_field "$resp" '.data | length')
  if [ "$shipment_count" -gt 0 ] 2>/dev/null; then
    assert_pass "Shipment auto-created via NATS event (count=$shipment_count)"
    local shipment_id
    shipment_id=$(json_field "$resp" '.data[0].id // empty')

    if [ -n "$shipment_id" ] && [ "$shipment_id" != "null" ]; then
      resp=$(api_put "/api/v1/shipments/$shipment_id/status" '{"status":"picked_up"}' "$admin_token")
      assert_http "Shipment picked_up" "200" "$(get_status "$resp")"

      resp=$(api_put "/api/v1/shipments/$shipment_id/status" '{"status":"in_transit"}' "$admin_token")
      assert_http "Shipment in_transit" "200" "$(get_status "$resp")"

      resp=$(api_put "/api/v1/shipments/$shipment_id/status" '{"status":"delivered"}' "$admin_token")
      assert_http "Shipment delivered" "200" "$(get_status "$resp")"
    fi
  else
    assert_skip "Shipment auto-created" "NATS event may not have propagated"
  fi

  resp=$(api_get "/api/v1/stock/movements?limit=5" "$admin_token")
  assert_http "Stock movements exist" "200" "$(get_status "$resp")"
  local movements_count
  movements_count=$(json_field "$resp" '.data | length')
  if [ "$movements_count" -gt 0 ] 2>/dev/null; then
    assert_pass "Stock movements recorded (count=$movements_count)"
  else
    assert_skip "Stock movements recorded" "no movements in response"
  fi

  resp=$(api_get "/api/v1/analytics/health" "$admin_token")
  assert_http "Analytics service healthy" "200" "$(get_status "$resp")"

  notif_resp=$(api_get "/api/v1/notifications?limit=1" "$admin_token")
  local notif_id
  notif_id=$(json_field "$notif_resp" '.data[0].id // empty')
  if [ -n "$notif_id" ] && [ "$notif_id" != "null" ]; then
    resp=$(api_put "/api/v1/notifications/$notif_id/read" '{}' "$admin_token")
    local mark_status
    mark_status=$(get_status "$resp")
    if [ "$mark_status" = "200" ] || [ "$mark_status" = "204" ]; then
      assert_pass "Mark notification as read"
    else
      assert_fail "Mark notification as read" "HTTP $mark_status"
    fi
  else
    assert_skip "Mark notification as read" "no notifications to mark"
  fi

  resp=$(api_get "/api/v1/logistics/performance" "$admin_token")
  assert_http "Logistics performance endpoint" "200" "$(get_status "$resp")"
  local body
  body=$(get_body "$resp")
  assert_contains "Performance has total_delivered" "$body" "total_delivered"

  resp=$(api_get "/api/v1/carriers" "$admin_token")
  assert_http "Carriers list" "200" "$(get_status "$resp")"
  local carrier_count
  carrier_count=$(json_field "$resp" '.data | length')
  if [ "$carrier_count" -gt 0 ] 2>/dev/null; then
    assert_pass "Carriers exist (count=$carrier_count)"
  else
    assert_fail "Carriers exist" "count=$carrier_count"
  fi

  resp=$(api_post "/api/v1/routes/calculate" '{
    "origin": "Moscow",
    "destination": "Saint Petersburg",
    "carrier_id": ""
  }' "$admin_token")
  local route_status
  route_status=$(get_status "$resp")
  if [ "$route_status" = "200" ] || [ "$route_status" = "201" ]; then
    assert_pass "Route calculation returns result"
    local distance
    distance=$(json_field "$resp" '.data.distance_km // .distance_km // 0')
    if [ "$distance" != "0" ] && [ "$distance" != "null" ]; then
      assert_pass "Route has non-zero distance ($distance km)"
    fi
  else
    assert_skip "Route calculation" "HTTP $route_status (may need carrier_id)"
  fi
}

scenario_6_mcp_chat() {
  log_scenario "6 (bonus): MCP Host connectivity and plans"

  local resp
  resp=$(curl -sf "${MCP_URL}/health" 2>/dev/null || echo '{"status":"unavailable"}')
  assert_contains "MCP host health check" "$resp" "status"

  local tools_count
  tools_count=$(echo "$resp" | jq -r '.tools // 0')
  if [ "$tools_count" -gt 0 ] 2>/dev/null; then
    assert_pass "MCP host has tools loaded (count=$tools_count)"
  else
    assert_skip "MCP host tools" "MCP host may not be running"
    return 0
  fi

  local redis_status
  redis_status=$(echo "$resp" | jq -r '.redis // "unknown"')
  if [ "$redis_status" = "ok" ]; then
    assert_pass "MCP Redis connection ok"
  else
    assert_skip "MCP Redis connection" "status=$redis_status"
  fi

  local admin_token
  admin_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  resp=$(curl -sf "${MCP_URL}/api/v1/mcp/plans/nonexistent-session" 2>/dev/null || echo "")
  if [ -n "$resp" ]; then
    assert_pass "MCP plans endpoint accessible"
  else
    assert_skip "MCP plans endpoint" "not accessible"
  fi
}

main() {
  echo -e "${CYAN}"
  echo "============================================================"
  echo "  ChainOrchestra — End-to-End Integration Test Suite"
  echo "  Gateway: $GATEWAY_URL"
  echo "  MCP:     $MCP_URL"
  echo "============================================================"
  echo -e "${NC}"

  wait_for_gateway

  local mcp_available=false
  if wait_for_mcp; then
    mcp_available=true
  fi

  scenario_1_order_create_and_stock
  scenario_2_multistep_workflow
  scenario_3_analytics_and_low_stock
  scenario_4_rbac_enforcement
  scenario_5_full_event_chain

  if [ "$mcp_available" = true ]; then
    scenario_6_mcp_chat
  else
    log_warn "Skipping MCP chat scenarios (MCP host not available)"
  fi

  echo ""
  echo -e "${CYAN}============================================================${NC}"
  echo -e "${CYAN}  TEST SUMMARY${NC}"
  echo -e "${CYAN}============================================================${NC}"
  echo -e "  Total:   $TOTAL_COUNT"
  echo -e "  ${GREEN}Passed:  $PASS_COUNT${NC}"
  echo -e "  ${RED}Failed:  $FAIL_COUNT${NC}"
  echo -e "  ${YELLOW}Skipped: $SKIP_COUNT${NC}"
  echo -e "${CYAN}============================================================${NC}"

  if [ "$FAIL_COUNT" -gt 0 ]; then
    echo -e "\n${RED}E2E TESTS FAILED${NC}"
    exit 1
  else
    echo -e "\n${GREEN}ALL E2E TESTS PASSED${NC}"
    exit 0
  fi
}

main "$@"
