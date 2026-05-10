#!/usr/bin/env bash
# ============================================================
# ChainOrchestra — Auth Integration E2E Test Suite (TASK-037)
# Tests full login -> gateway -> services flow with RBAC.
# Requires: running docker compose stack + seed data
# Usage: ./scripts/e2e-auth-test.sh
# ============================================================
set -uo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@chainorchestra.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"
OPERATOR_EMAIL="ivan.petrenko@chainorchestra.local"
OPERATOR_PASSWORD="Operator1!"
WAREHOUSE_EMAIL="maria.kovalenko@chainorchestra.local"
WAREHOUSE_PASSWORD="Warehouse1!"
LOGISTICS_EMAIL="oleksii.shevchenko@chainorchestra.local"
LOGISTICS_PASSWORD="Logistics1!"
ANALYST_EMAIL="olena.bondarenko@chainorchestra.local"
ANALYST_PASSWORD="Analyst1!"

# Colors
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

# ---------- helpers ----------

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

# HTTP helpers
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

login_full() {
  local email="$1"
  local password="$2"
  api_post "/api/v1/auth/login" "{\"email\":\"$email\",\"password\":\"$password\"}"
}

# ============================================================
# Wait for gateway
# ============================================================
log_step "Waiting for API Gateway"
for i in $(seq 1 30); do
  if curl -sf "${GATEWAY_URL}/health" > /dev/null 2>&1; then
    log_info "Gateway is ready"
    break
  fi
  if [ "$i" -eq 30 ]; then
    log_error "Gateway not ready after 30s"
    exit 1
  fi
  sleep 1
done

# ============================================================
# SCENARIO 1: Login via gateway -> JWT with user_id, email, role
# ============================================================
scenario_1_login_jwt() {
  log_scenario "1: Login via gateway -> JWT with claims"

  # Step 1: Login as admin through gateway
  local resp
  resp=$(login_full "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  local status
  status=$(get_status "$resp")
  assert_http "Admin login via gateway" "200" "$status"

  local access_token
  access_token=$(json_field "$resp" '.data.access_token // empty')
  assert_not_empty "Access token returned" "$access_token"

  local refresh_token
  refresh_token=$(json_field "$resp" '.data.refresh_token // empty')
  assert_not_empty "Refresh token returned" "$refresh_token"

  # Step 2: Decode JWT and verify claims (user_id, email, role)
  local payload
  payload=$(echo "$access_token" | cut -d. -f2 | base64 -d 2>/dev/null || echo "$access_token" | cut -d. -f2 | base64 -D 2>/dev/null || echo "")
  if [ -n "$payload" ]; then
    local jwt_user_id
    jwt_user_id=$(echo "$payload" | jq -r '.user_id // empty' 2>/dev/null)
    assert_not_empty "JWT contains user_id" "$jwt_user_id"

    local jwt_email
    jwt_email=$(echo "$payload" | jq -r '.email // empty' 2>/dev/null)
    assert_eq "JWT contains correct email" "$ADMIN_EMAIL" "$jwt_email"

    local jwt_role
    jwt_role=$(echo "$payload" | jq -r '.role // empty' 2>/dev/null)
    assert_eq "JWT contains correct role" "admin" "$jwt_role"
  else
    assert_skip "JWT decode" "base64 decode failed on this platform"
  fi

  # Step 3: Login as each role and verify JWT claims
  for role_info in "operator:$OPERATOR_EMAIL:$OPERATOR_PASSWORD" \
                   "warehouse_manager:$WAREHOUSE_EMAIL:$WAREHOUSE_PASSWORD" \
                   "logistics_manager:$LOGISTICS_EMAIL:$LOGISTICS_PASSWORD" \
                   "analyst:$ANALYST_EMAIL:$ANALYST_PASSWORD"; do
    local role email password
    role=$(echo "$role_info" | cut -d: -f1)
    email=$(echo "$role_info" | cut -d: -f2)
    password=$(echo "$role_info" | cut -d: -f3)

    resp=$(login_full "$email" "$password")
    status=$(get_status "$resp")
    assert_http "Login as $role" "200" "$status"
  done

  # Step 4: Login with invalid credentials -> 401
  resp=$(login_full "$ADMIN_EMAIL" "wrong_password")
  status=$(get_status "$resp")
  assert_http "Invalid password returns 401" "401" "$status"

  resp=$(login_full "nonexistent@test.com" "pass123")
  status=$(get_status "$resp")
  assert_http "Unknown email returns 401" "401" "$status"
}

# ============================================================
# SCENARIO 2: Gateway proxies with X-User-ID, X-User-Role headers
# ============================================================
scenario_2_gateway_headers() {
  log_scenario "2: Gateway proxies requests with X-User-* headers"

  local admin_token
  admin_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  assert_not_empty "Admin login for header test" "$admin_token"
  if [ -z "$admin_token" ]; then return 1; fi

  # Step 1: GET /api/v1/users/me should return current user (proves X-User-ID was forwarded)
  local resp
  resp=$(api_get "/api/v1/users/me" "$admin_token")
  local status
  status=$(get_status "$resp")
  assert_http "Profile via gateway (X-User-ID forwarded)" "200" "$status"

  local profile_email
  profile_email=$(json_field "$resp" '.data.email // empty')
  assert_eq "Profile returns correct email" "$ADMIN_EMAIL" "$profile_email"

  local profile_role
  profile_role=$(json_field "$resp" '.data.role // empty')
  assert_eq "Profile returns correct role" "admin" "$profile_role"

  # Step 2: GET /api/v1/users (admin-only) proves X-User-Role forwarded
  resp=$(api_get "/api/v1/users?limit=10" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin list users (X-User-Role=admin forwarded)" "200" "$status"

  # Step 3: Verify each service responds through gateway
  resp=$(api_get "/api/v1/orders?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Orders via gateway" "200" "$status"

  resp=$(api_get "/api/v1/products?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Products via gateway" "200" "$status"

  resp=$(api_get "/api/v1/shipments?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Shipments via gateway" "200" "$status"

  resp=$(api_get "/api/v1/analytics/sales/summary?from=2020-01-01&to=2030-12-31" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Analytics via gateway" "200" "$status"

  resp=$(api_get "/api/v1/notifications?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Notifications via gateway" "200" "$status"
}

# ============================================================
# SCENARIO 3: RBAC enforcement across all services
# ============================================================
scenario_3_rbac_enforcement() {
  log_scenario "3: RBAC enforcement (role-based access per service)"

  # -- No token -> 401 --
  local resp status
  resp=$(api_get "/api/v1/users/me")
  status=$(get_status "$resp")
  assert_http "No token -> 401" "401" "$status"

  # -- Invalid token -> 401 --
  resp=$(api_get "/api/v1/users/me" "invalid-jwt-token-xyz")
  status=$(get_status "$resp")
  assert_http "Invalid token -> 401" "401" "$status"

  # -- Operator: can access orders, notifications; cannot access users, inventory --
  local op_token
  op_token=$(login "$OPERATOR_EMAIL" "$OPERATOR_PASSWORD")
  if [ -z "$op_token" ]; then
    assert_skip "Operator RBAC" "cannot login as operator"
    return 1
  fi

  resp=$(api_get "/api/v1/orders?limit=1" "$op_token")
  status=$(get_status "$resp")
  assert_http "Operator can list orders" "200" "$status"

  resp=$(api_get "/api/v1/users/me" "$op_token")
  status=$(get_status "$resp")
  assert_http "Operator can access own profile" "200" "$status"

  local op_role
  op_role=$(json_field "$resp" '.data.role // empty')
  assert_eq "Operator role is correct" "operator" "$op_role"

  resp=$(api_get "/api/v1/users" "$op_token")
  status=$(get_status "$resp")
  assert_http "Operator CANNOT list users (admin-only)" "403" "$status"

  # -- Operator cannot escalate own role --
  resp=$(api_put "/api/v1/users/me" '{"role":"admin"}' "$op_token")
  resp=$(api_get "/api/v1/users/me" "$op_token")
  op_role=$(json_field "$resp" '.data.role // empty')
  assert_eq "Operator cannot escalate role" "operator" "$op_role"

  # -- Warehouse manager: can access products/stock/warehouses --
  local wh_token
  wh_token=$(login "$WAREHOUSE_EMAIL" "$WAREHOUSE_PASSWORD")
  if [ -n "$wh_token" ]; then
    resp=$(api_get "/api/v1/products?limit=1" "$wh_token")
    status=$(get_status "$resp")
    assert_http "Warehouse manager can list products" "200" "$status"

    resp=$(api_get "/api/v1/stock?limit=1" "$wh_token")
    status=$(get_status "$resp")
    assert_http "Warehouse manager can list stock" "200" "$status"

    resp=$(api_get "/api/v1/warehouses?limit=1" "$wh_token")
    status=$(get_status "$resp")
    assert_http "Warehouse manager can list warehouses" "200" "$status"

    resp=$(api_get "/api/v1/users" "$wh_token")
    status=$(get_status "$resp")
    assert_http "Warehouse manager CANNOT list users" "403" "$status"
  else
    assert_skip "Warehouse manager RBAC" "cannot login"
  fi

  # -- Logistics manager: can access shipments/carriers --
  local log_token
  log_token=$(login "$LOGISTICS_EMAIL" "$LOGISTICS_PASSWORD")
  if [ -n "$log_token" ]; then
    resp=$(api_get "/api/v1/shipments?limit=1" "$log_token")
    status=$(get_status "$resp")
    assert_http "Logistics manager can list shipments" "200" "$status"

    resp=$(api_get "/api/v1/carriers?limit=1" "$log_token")
    status=$(get_status "$resp")
    assert_http "Logistics manager can list carriers" "200" "$status"

    resp=$(api_get "/api/v1/users" "$log_token")
    status=$(get_status "$resp")
    assert_http "Logistics manager CANNOT list users" "403" "$status"
  else
    assert_skip "Logistics manager RBAC" "cannot login"
  fi

  # -- Analyst: can access analytics --
  local an_token
  an_token=$(login "$ANALYST_EMAIL" "$ANALYST_PASSWORD")
  if [ -n "$an_token" ]; then
    resp=$(api_get "/api/v1/analytics/sales/summary?from=2020-01-01&to=2030-12-31" "$an_token")
    status=$(get_status "$resp")
    assert_http "Analyst can access analytics" "200" "$status"

    resp=$(api_get "/api/v1/users" "$an_token")
    status=$(get_status "$resp")
    assert_http "Analyst CANNOT list users" "403" "$status"
  else
    assert_skip "Analyst RBAC" "cannot login"
  fi

  # -- Admin: can access everything --
  local admin_token
  admin_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  resp=$(api_get "/api/v1/users?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list users" "200" "$status"

  resp=$(api_get "/api/v1/orders?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list orders" "200" "$status"

  resp=$(api_get "/api/v1/products?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list products" "200" "$status"

  resp=$(api_get "/api/v1/shipments?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list shipments" "200" "$status"

  resp=$(api_get "/api/v1/analytics/sales/summary?from=2020-01-01&to=2030-12-31" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can access analytics" "200" "$status"

  resp=$(api_get "/api/v1/notifications?limit=1" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Admin can list notifications" "200" "$status"
}

# ============================================================
# SCENARIO 4: Refresh token flow
# ============================================================
scenario_4_refresh_token() {
  log_scenario "4: Refresh token flow"

  local resp
  resp=$(login_full "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  local status
  status=$(get_status "$resp")
  assert_http "Login for refresh test" "200" "$status"

  local refresh_token
  refresh_token=$(json_field "$resp" '.data.refresh_token // empty')
  assert_not_empty "Refresh token in login response" "$refresh_token"
  if [ -z "$refresh_token" ] || [ "$refresh_token" = "null" ]; then
    assert_skip "Refresh token flow" "no refresh_token in login response"
    return 1
  fi

  # Step 1: Use refresh token to get new access token
  resp=$(api_post "/api/v1/auth/refresh" "{\"refresh_token\":\"$refresh_token\"}")
  status=$(get_status "$resp")
  assert_http "Refresh token -> new access token" "200" "$status"

  local new_access
  new_access=$(json_field "$resp" '.data.access_token // empty')
  assert_not_empty "New access token returned" "$new_access"

  local new_refresh
  new_refresh=$(json_field "$resp" '.data.refresh_token // empty')
  assert_not_empty "New refresh token returned" "$new_refresh"

  # Step 2: Verify new access token works
  resp=$(api_get "/api/v1/users/me" "$new_access")
  status=$(get_status "$resp")
  assert_http "New access token works for profile" "200" "$status"

  local profile_email
  profile_email=$(json_field "$resp" '.data.email // empty')
  assert_eq "Profile email with new token" "$ADMIN_EMAIL" "$profile_email"

  # Step 3: Use new access token on other services
  resp=$(api_get "/api/v1/orders?limit=1" "$new_access")
  status=$(get_status "$resp")
  assert_http "New token works for orders" "200" "$status"

  resp=$(api_get "/api/v1/products?limit=1" "$new_access")
  status=$(get_status "$resp")
  assert_http "New token works for products" "200" "$status"

  # Step 4: Invalid refresh token -> error
  resp=$(api_post "/api/v1/auth/refresh" '{"refresh_token":"invalid-token-xyz"}')
  status=$(get_status "$resp")
  if [ "$status" = "401" ] || [ "$status" = "400" ]; then
    assert_pass "Invalid refresh token rejected (HTTP $status)"
  else
    assert_fail "Invalid refresh token rejected" "expected 401 or 400, got $status"
  fi
}

# ============================================================
# SCENARIO 5: Expired/invalid token -> 401
# ============================================================
scenario_5_expired_token() {
  log_scenario "5: Expired/invalid token handling"

  # Craft an obviously expired JWT (just random base64 — gateway should reject)
  local fake_jwt="eyJhbGciOiJIUzI1NiJ9.eyJleHAiOjEwMDAwMDAwMDB9.invalid-sig"
  local resp status

  resp=$(api_get "/api/v1/users/me" "$fake_jwt")
  status=$(get_status "$resp")
  assert_http "Expired/invalid JWT -> 401" "401" "$status"

  # Empty bearer
  resp=$(api_get "/api/v1/orders" "")
  status=$(get_status "$resp")
  assert_http "Empty token -> 401" "401" "$status"

  # Malformed token
  resp=$(api_get "/api/v1/products" "not.a.jwt")
  status=$(get_status "$resp")
  assert_http "Malformed token -> 401" "401" "$status"
}

# ============================================================
# SCENARIO 6: Password reset flow
# ============================================================
scenario_6_password_reset() {
  log_scenario "6: Password reset flow"

  local resp status

  # Request password reset (should always return 200 — no email enumeration)
  resp=$(api_post "/api/v1/auth/password-reset" "{\"email\":\"$OPERATOR_EMAIL\"}")
  status=$(get_status "$resp")
  assert_http "Password reset request accepted" "200" "$status"

  # Non-existent email — still 200
  resp=$(api_post "/api/v1/auth/password-reset" '{"email":"nonexistent@test.com"}')
  status=$(get_status "$resp")
  assert_http "Password reset no email enumeration" "200" "$status"

  # Confirm with invalid token
  resp=$(api_post "/api/v1/auth/password-reset/confirm" '{"token":"invalid","new_password":"new123"}')
  status=$(get_status "$resp")
  assert_http "Invalid reset token rejected" "400" "$status"
}

# ============================================================
# SCENARIO 7: Full chain — Frontend -> Gateway -> Service -> Response
# ============================================================
scenario_7_full_chain() {
  log_scenario "7: Full chain (login -> gateway -> multiple services)"

  # Login as admin
  local admin_token
  admin_token=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
  assert_not_empty "Full chain: admin login" "$admin_token"
  if [ -z "$admin_token" ]; then return 1; fi

  # Create an order through gateway
  local resp status
  resp=$(api_post "/api/v1/orders" '{
    "customer_name": "Auth E2E Customer",
    "customer_email": "auth-e2e@test.com",
    "shipping_address": "Test St, 1, Kyiv",
    "items": [
      {"product_name":"Auth Widget","quantity":5,"unit_price":100}
    ]
  }' "$admin_token")
  status=$(get_status "$resp")
  assert_http "Create order through full chain" "201" "$status"

  local order_id
  order_id=$(json_field "$resp" '.data.id // empty')
  assert_not_empty "Order created with ID" "$order_id"

  # Verify order is retrievable
  if [ -n "$order_id" ] && [ "$order_id" != "null" ]; then
    resp=$(api_get "/api/v1/orders/$order_id" "$admin_token")
    status=$(get_status "$resp")
    assert_http "Retrieve created order" "200" "$status"

    local customer
    customer=$(json_field "$resp" '.data.customer_name // empty')
    assert_eq "Order customer matches" "Auth E2E Customer" "$customer"
  fi

  # Create notification through gateway (admin-only)
  resp=$(api_post "/api/v1/notifications" '{
    "user_id": "'"$(json_field "$(api_get "/api/v1/users/me" "$admin_token")" '.data.id // empty')"'",
    "type": "system",
    "title": "Auth E2E Test",
    "message": "Testing full auth chain"
  }' "$admin_token")
  status=$(get_status "$resp")
  assert_http "Create notification (admin)" "201" "$status"

  # Verify unread count
  resp=$(api_get "/api/v1/notifications/unread-count" "$admin_token")
  status=$(get_status "$resp")
  assert_http "Unread count via full chain" "200" "$status"

  local count
  count=$(json_field "$resp" '.data.unread_count // .unread_count // 0')
  if [ "$count" -ge 1 ] 2>/dev/null; then
    assert_pass "Unread count >= 1 ($count)"
  else
    assert_fail "Unread count >= 1" "got: $count"
  fi

  # Verify operator sees the same order but cannot create notifications
  local op_token
  op_token=$(login "$OPERATOR_EMAIL" "$OPERATOR_PASSWORD")
  if [ -n "$op_token" ] && [ -n "$order_id" ] && [ "$order_id" != "null" ]; then
    resp=$(api_get "/api/v1/orders/$order_id" "$op_token")
    status=$(get_status "$resp")
    assert_http "Operator can see order" "200" "$status"

    resp=$(api_post "/api/v1/notifications" '{
      "user_id":"test","type":"system","title":"Test","message":"msg"
    }' "$op_token")
    status=$(get_status "$resp")
    assert_http "Operator CANNOT create notification" "403" "$status"
  fi
}

# ============================================================
# Run all scenarios
# ============================================================
log_step "ChainOrchestra Auth Integration E2E Tests (TASK-037)"
echo ""

scenario_1_login_jwt
scenario_2_gateway_headers
scenario_3_rbac_enforcement
scenario_4_refresh_token
scenario_5_expired_token
scenario_6_password_reset
scenario_7_full_chain

# ============================================================
# Summary
# ============================================================
echo ""
log_step "Results"
echo -e "  Total:   $TOTAL_COUNT"
echo -e "  ${GREEN}Passed:  $PASS_COUNT${NC}"
echo -e "  ${RED}Failed:  $FAIL_COUNT${NC}"
echo -e "  ${YELLOW}Skipped: $SKIP_COUNT${NC}"
echo ""

if [ "$FAIL_COUNT" -gt 0 ]; then
  log_error "Some tests failed!"
  exit 1
fi

log_info "All auth integration tests passed!"
exit 0
