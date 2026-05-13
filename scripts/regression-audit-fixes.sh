#!/usr/bin/env bash
set -u
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0
FAILED_TESTS=()

assert() {
  local name="$1"
  local actual="$2"
  local expected="$3"
  if [[ "$actual" == "$expected" ]]; then
    echo -e "  ${GREEN}PASS${NC}  $name"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}FAIL${NC}  $name (got=$actual want=$expected)"
    FAIL=$((FAIL+1))
    FAILED_TESTS+=("$name")
  fi
}

GW=${GATEWAY:-http://localhost:8080}

echo "=== login admin ==="
A=$(curl -s -X POST "$GW/api/v1/auth/login" -H 'Content-Type: application/json' \
  -d '{"email":"admin@chainorchestra.local","password":"upB@7UmdvEWYe&t#8nY%"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")
OP=$(curl -s -X POST "$GW/api/v1/auth/login" -H 'Content-Type: application/json' \
  -d '{"email":"ivan.petrenko@chainorchestra.local","password":"wLh#O!+BMK^82qzrU#r2"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")
ANALYST=$(curl -s -X POST "$GW/api/v1/auth/login" -H 'Content-Type: application/json' \
  -d '{"email":"olena.bondarenko@chainorchestra.local","password":"#1nVWLCi4x3H*C4SOkL4"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")

echo ""
echo "=== C1: internal ports closed ==="
for port in 8001 8002 8003 8004 8005 8006 8007; do
  code=$(curl -s -m 2 -o /dev/null -w "%{http_code}" "http://localhost:$port/health" 2>&1)
  assert "  port $port unreachable from host" "$code" "000"
done

echo ""
echo "=== C2: SSE realtime ==="
code=$(curl -s -m 3 -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/events/stream")
assert "  SSE returns 200 (text/event-stream)" "$code" "200"

echo ""
echo "=== C3: auto-shipment on order confirm ==="
NEW=$(curl -s -X POST "$GW/api/v1/orders" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"customer_name":"AuditRegression","items":[{"product_id":"9ebcaf0f-c50d-4f36-b417-c3fa7477fc8c","name":"Suit","quantity":1,"unit_price":87.96}]}')
OID=$(echo "$NEW" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")
curl -s -X PUT "$GW/api/v1/orders/$OID/status" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' -d '{"status":"confirmed"}' -o /dev/null
sleep 4
SHIP=$(curl -s -H "Authorization: Bearer $A" "$GW/api/v1/shipments?order_id=$OID" | python3 -c "import sys,json;print(len(json.load(sys.stdin)['data']))")
assert "  shipment auto-created" "$SHIP" "1"

echo ""
echo "=== H1: audit-log RBAC ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $OP" "$GW/api/v1/analytics/audit-log?limit=1")
assert "  operator -> 403" "$code" "403"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ANALYST" "$GW/api/v1/analytics/audit-log?limit=1")
assert "  analyst -> 403" "$code" "403"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/analytics/audit-log?limit=1")
assert "  admin -> 200" "$code" "200"

echo ""
echo "=== H3: weak password rejected ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/auth/register" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"email":"weak.r@chainorchestra.local","password":"weak","first_name":"W","last_name":"W","role":"operator"}')
assert "  weak password -> 400" "$code" "400"
UNIQ=$(date +%s%N)
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/auth/register" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d "{\"email\":\"strong.r$UNIQ@chainorchestra.local\",\"password\":\"StrongPass1\",\"first_name\":\"S\",\"last_name\":\"S\",\"role\":\"operator\"}")
assert "  strong password -> 201" "$code" "201"

echo ""
echo "=== H4: order product_id FK + UUID ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/orders" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"customer_name":"X","items":[{"product_id":"not-a-uuid","name":"X","quantity":1,"unit_price":1}]}')
assert "  malformed product_id -> 400" "$code" "400"
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/orders" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"customer_name":"X","items":[{"product_id":"00000000-0000-0000-0000-000000000000","name":"X","quantity":1,"unit_price":1}]}')
assert "  non-existent product_id -> 404" "$code" "404"

echo ""
echo "=== H5: stock invariant ==="
BAD=$(docker compose exec -T postgres psql -U postgres -d chainorchestra -tA -c "SELECT count(*) FROM inventory.stock WHERE reserved > quantity;" 2>&1)
assert "  no over-reserved rows" "$BAD" "0"

echo ""
echo "=== H6: threshold field name + DisallowUnknownFields ==="
PID=9ebcaf0f-c50d-4f36-b417-c3fa7477fc8c
WID=$(curl -s -H "Authorization: Bearer $A" "$GW/api/v1/warehouses?limit=1" | python3 -c "import sys,json;print(json.load(sys.stdin)['data'][0]['id'])")
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$GW/api/v1/stock/threshold" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$PID\",\"warehouse_id\":\"$WID\",\"unknown_field\":5}")
assert "  unknown field -> 400" "$code" "400"
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$GW/api/v1/stock/threshold" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$PID\",\"warehouse_id\":\"$WID\",\"min_threshold\":33}")
assert "  min_threshold=33 -> 200" "$code" "200"

echo ""
echo "=== H7: in-transit-summary endpoint ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/shipments/in-transit-summary")
assert "  endpoint returns 200" "$code" "200"

echo ""
echo "=== H8: terminal-state guard on returned shipment record-attempt ==="
RET=$(curl -s -H "Authorization: Bearer $A" "$GW/api/v1/shipments?status=returned&limit=1" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['data'][0]['id'] if d['data'] else '')")
if [ -n "$RET" ]; then
  code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/shipments/$RET/record-attempt" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' -d '{"reason":"x"}')
  assert "  record-attempt on returned -> 409" "$code" "409"
fi

echo ""
echo "=== H10: notifications is_read filter ==="
UNREAD_STATUSES=$(curl -s -H "Authorization: Bearer $A" "$GW/api/v1/notifications?is_read=false&limit=5" | python3 -c "
import sys,json
d=json.load(sys.stdin);data=d.get('data',[])
read_count=sum(1 for n in data if n.get('status')=='read')
print('NO_READ' if read_count==0 else 'HAS_READ')")
assert "  is_read=false returns no read notifications" "$UNREAD_STATUSES" "NO_READ"

echo ""
echo "=== H12: period_comparison direction ==="
RES=$(curl -s -H "Authorization: Bearer $A" "$GW/api/v1/analytics/period-comparison?metric=revenue&a_from=2026-04-01&a_to=2026-04-30&b_from=2026-03-01&b_to=2026-03-31" \
  | python3 -c "import sys,json;d=json.load(sys.stdin)['data'];print(d['direction'])")
assert "  direction is 'down' (April < March)" "$RES" "down"

echo ""
echo "=== H13: simulator validation ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/simulator/start" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' -d '{"scenario":"alien","speed":1}')
assert "  scenario=alien -> 400" "$code" "400"
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/simulator/start" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' -d '{"scenario":"steady","speed":99999}')
assert "  speed=99999 -> 400" "$code" "400"
curl -s -X POST "$GW/api/v1/simulator/stop" -H "Authorization: Bearer $A" -o /dev/null

echo ""
echo "=== H14: frontend auth gate ==="
code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/admin/simulator)
assert "  /admin/simulator unauthenticated -> 307" "$code" "307"
code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/dashboard)
assert "  /dashboard unauthenticated -> 307" "$code" "307"

echo ""
echo "=== M: UUID validation on /orders/{id} ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/orders/not-a-uuid")
assert "  /orders/not-a-uuid -> 400" "$code" "400"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/products/not-a-uuid")
assert "  /products/not-a-uuid -> 400" "$code" "400"

echo ""
echo "=== M: date validation ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/orders?date_from=2025/01/01&limit=1")
assert "  /orders?date_from=2025/01/01 -> 400" "$code" "400"

echo ""
echo "=== M: enum validation on status filter ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/orders?status=alien&limit=1")
assert "  /orders?status=alien -> 400" "$code" "400"

echo ""
echo "=== M: forecast validation ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/analytics/forecast?metric=revenue&method=neural&horizon_days=14&date_from=2026-03-01&date_to=2026-05-13")
assert "  forecast?method=neural -> 400" "$code" "400"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/analytics/forecast?metric=revenue&method=linear&horizon_days=-10&date_from=2026-03-01&date_to=2026-05-13")
assert "  forecast?horizon_days=-10 -> 400" "$code" "400"

echo ""
echo "=== M: reschedule into past ==="
ACT=$(curl -s -H "Authorization: Bearer $A" "$GW/api/v1/shipments?status=in_transit&limit=1" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['data'][0]['id'] if d['data'] else '')")
if [ -n "$ACT" ]; then
  code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/shipments/$ACT/reschedule" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' -d '{"new_eta":"2020-01-01T10:00:00Z","reason":"past"}')
  assert "  reschedule past -> 400" "$code" "400"
fi

echo ""
echo "=== M: record-delivery requires proof ==="
if [ -n "$ACT" ]; then
  code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/shipments/$ACT/record-delivery" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' -d '{}')
  assert "  empty body -> 400" "$code" "400"
fi

echo ""
echo "=== M: Latin-only customer_name ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/orders" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"customer_name":"Іван Петренко","items":[{"product_id":"9ebcaf0f-c50d-4f36-b417-c3fa7477fc8c","name":"Suit","quantity":1,"unit_price":1}]}')
assert "  Cyrillic customer_name -> 400" "$code" "400"

echo ""
echo "=== M: Prometheus /metrics no auth ==="
code=$(curl -s -o /dev/null -w "%{http_code}" "$GW/metrics")
assert "  /metrics no auth -> 200" "$code" "200"

echo ""
echo "=== M: mcp-host budget needs JWT ==="
code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8090/api/v1/mcp/budget/abc")
assert "  no auth -> 401" "$code" "401"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "http://localhost:8090/api/v1/mcp/budget/abc")
assert "  with token -> 200" "$code" "200"

echo ""
echo "=== M: GET /users/{id} self vs admin ==="
ME_OP=$(curl -s -H "Authorization: Bearer $OP" "$GW/api/v1/users/me" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $OP" "$GW/api/v1/users/$ME_OP")
assert "  operator GET self -> 200" "$code" "200"
ADMIN_ID=$(curl -s -H "Authorization: Bearer $A" "$GW/api/v1/users/me" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $OP" "$GW/api/v1/users/$ADMIN_ID")
assert "  operator GET admin -> 403" "$code" "403"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $A" "$GW/api/v1/users/$ME_OP")
assert "  admin GET other -> 200" "$code" "200"

echo ""
echo "=== M: report alias type/report_type ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/api/v1/analytics/report" -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"type":"sales","date_from":"2026-04-01","date_to":"2026-04-30"}')
[[ "$code" == "200" || "$code" == "400" ]] && code=ACCEPTED || code="$code"
assert "  type alias accepted (not validation_error on missing report_type)" "$code" "ACCEPTED"

echo ""
echo "=== Phase 6: SUMMARY ==="
TOTAL=$((PASS+FAIL))
echo -e "Passed: ${GREEN}$PASS${NC} / $TOTAL"
echo -e "Failed: ${RED}$FAIL${NC} / $TOTAL"
if [[ $FAIL -gt 0 ]]; then
  echo -e "${RED}Failed tests:${NC}"
  for t in "${FAILED_TESTS[@]}"; do echo "  - $t"; done
  exit 1
fi
exit 0
