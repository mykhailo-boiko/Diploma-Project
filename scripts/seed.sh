#!/usr/bin/env bash
# ============================================
# ChainOrchestra — Seed Data Script
# Populates the system with realistic test data
# Idempotent: safe to run multiple times
# ============================================
set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@chainorchestra.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "\n${BLUE}=== $1 ===${NC}"; }

# ---------- helpers ----------

api_post() {
  local path="$1"
  local data="$2"
  local token="${3:-}"
  local headers=(-s -w "\n%{http_code}" -H "Content-Type: application/json")
  if [ -n "$token" ]; then
    headers+=(-H "Authorization: Bearer $token")
  fi
  curl "${headers[@]}" -X POST "${GATEWAY_URL}${path}" -d "$data"
}

api_get() {
  local path="$1"
  local token="${2:-}"
  local headers=(-s -w "\n%{http_code}" -H "Content-Type: application/json")
  if [ -n "$token" ]; then
    headers+=(-H "Authorization: Bearer $token")
  fi
  curl "${headers[@]}" "${GATEWAY_URL}${path}"
}

api_put() {
  local path="$1"
  local data="$2"
  local token="${3:-}"
  local headers=(-s -w "\n%{http_code}" -H "Content-Type: application/json")
  if [ -n "$token" ]; then
    headers+=(-H "Authorization: Bearer $token")
  fi
  curl "${headers[@]}" -X PUT "${GATEWAY_URL}${path}" -d "$data"
}

# Parse response: last line is HTTP status code, rest is body
parse_response() {
  local response="$1"
  local body
  body=$(echo "$response" | sed '$d')
  local status
  status=$(echo "$response" | tail -1)
  echo "$body"
  return 0
}

get_status() {
  local response="$1"
  echo "$response" | tail -1
}

# Extract field from JSON (simple jq wrapper)
json_field() {
  echo "$1" | sed '$d' | jq -r "$2"
}

# ---------- wait for gateway ----------

wait_for_gateway() {
  log_step "Waiting for API Gateway"
  local max_retries=30
  local retry=0
  while [ $retry -lt $max_retries ]; do
    if curl -sf "${GATEWAY_URL}/health" > /dev/null 2>&1; then
      log_info "API Gateway is ready at ${GATEWAY_URL}"
      return 0
    fi
    retry=$((retry + 1))
    echo -n "."
    sleep 2
  done
  log_error "API Gateway not available after ${max_retries} retries"
  exit 1
}

# ---------- authenticate ----------

admin_login() {
  log_step "Authenticating as admin"
  local response
  response=$(api_post "/api/v1/auth/login" "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}")
  local status
  status=$(get_status "$response")
  if [ "$status" != "200" ]; then
    log_error "Admin login failed (HTTP $status)"
    echo "$response" | sed '$d'
    exit 1
  fi
  ADMIN_TOKEN=$(json_field "$response" '.data.access_token')
  log_info "Admin authenticated (token: ${ADMIN_TOKEN:0:20}...)"
}

# ---------- 1. Users ----------

declare -A USER_TOKENS
declare -A USER_IDS

create_users() {
  log_step "Creating Users (5 users, one per role)"

  local users=(
    '{"email":"ivan.petrenko@chainorchestra.local","password":"wLh#O!+BMK^82qzrU#r2","first_name":"Ivan","last_name":"Petrenko","role":"operator"}'
    '{"email":"maria.kovalenko@chainorchestra.local","password":"+5=SB8#Yd0QYHlIJnAcU","first_name":"Mariia","last_name":"Kovalenko","role":"warehouse_manager"}'
    '{"email":"oleksii.shevchenko@chainorchestra.local","password":"F4EJ%%88TO4e1IqdbD4S","first_name":"Oleksii","last_name":"Shevchenko","role":"logistics_manager"}'
    '{"email":"olena.bondarenko@chainorchestra.local","password":"#1nVWLCi4x3H*C4SOkL4","first_name":"Olena","last_name":"Bondarenko","role":"analyst"}'
  )

  local emails=(
    "ivan.petrenko@chainorchestra.local"
    "maria.kovalenko@chainorchestra.local"
    "oleksii.shevchenko@chainorchestra.local"
    "olena.bondarenko@chainorchestra.local"
  )

  local roles=(operator warehouse_manager logistics_manager analyst)

  for i in "${!users[@]}"; do
    local email="${emails[$i]}"
    local role="${roles[$i]}"
    local response
    response=$(api_post "/api/v1/auth/register" "${users[$i]}" "$ADMIN_TOKEN")
    local status
    status=$(get_status "$response")
    if [ "$status" = "201" ] || [ "$status" = "200" ]; then
      log_info "Created user: $email ($role)"
    elif [ "$status" = "409" ] || [ "$status" = "400" ]; then
      log_warn "User $email already exists, skipping"
    else
      log_error "Failed to create user $email (HTTP $status)"
      echo "$response" | sed '$d'
    fi
  done

  # Login as each user to get tokens and IDs
  log_info "Logging in as each user to get tokens..."

  local all_emails=("$ADMIN_EMAIL" "${emails[@]}")
  local all_passwords=("$ADMIN_PASSWORD" "wLh#O!+BMK^82qzrU#r2" "+5=SB8#Yd0QYHlIJnAcU" "F4EJ%%88TO4e1IqdbD4S" "#1nVWLCi4x3H*C4SOkL4")
  local all_roles=("admin" "${roles[@]}")

  for i in "${!all_emails[@]}"; do
    local email="${all_emails[$i]}"
    local password="${all_passwords[$i]}"
    local role="${all_roles[$i]}"
    local response
    response=$(api_post "/api/v1/auth/login" "{\"email\":\"${email}\",\"password\":\"${password}\"}")
    local status
    status=$(get_status "$response")
    if [ "$status" = "200" ]; then
      local token
      token=$(json_field "$response" '.data.access_token')
      USER_TOKENS["$role"]="$token"

      # Get user ID from /users/me
      local me_response
      me_response=$(api_get "/api/v1/users/me" "$token")
      local user_id
      user_id=$(json_field "$me_response" '.data.id')
      USER_IDS["$role"]="$user_id"
      log_info "  $role ($email) — ID: ${user_id:0:8}..."
    else
      log_error "Failed to login as $email"
    fi
  done
}

# ---------- 2. Products ----------

declare -a PRODUCT_IDS=()

create_products() {
  log_step "Creating Products (20 products in 5 categories)"

  local products=(
    '{"sku":"ELEC-LAPTOP-001","name":"ProBook Laptop 15","description":"15.6 inch business laptop with 16GB RAM and 512GB SSD","category":"Electronics","unit_price":899.99}'
    '{"sku":"ELEC-PHONE-001","name":"SmartPhone X12","description":"Flagship smartphone with OLED display and 128GB storage","category":"Electronics","unit_price":699.99}'
    '{"sku":"ELEC-TABLET-001","name":"TabPro 10","description":"10.1 inch tablet with stylus support and 256GB storage","category":"Electronics","unit_price":449.99}'
    '{"sku":"ELEC-HEADPH-001","name":"NoiseFree Pro Headphones","description":"Active noise cancelling wireless headphones","category":"Electronics","unit_price":199.99}'
    '{"sku":"FURN-CHAIR-001","name":"ErgoMax Office Chair","description":"Ergonomic mesh office chair with lumbar support","category":"Furniture","unit_price":349.99}'
    '{"sku":"FURN-DESK-001","name":"StandUp Adjustable Desk","description":"Electric height-adjustable standing desk 160x80cm","category":"Furniture","unit_price":549.99}'
    '{"sku":"FURN-SHELF-001","name":"BookWall Shelf Unit","description":"5-tier modular bookshelf in oak finish","category":"Furniture","unit_price":179.99}'
    '{"sku":"FURN-CABINET-001","name":"SecureLock Filing Cabinet","description":"3-drawer steel filing cabinet with lock","category":"Furniture","unit_price":229.99}'
    '{"sku":"CLOTH-JACKET-001","name":"AllWeather Jacket","description":"Waterproof breathable jacket for outdoor use","category":"Clothing","unit_price":129.99}'
    '{"sku":"CLOTH-SHIRT-001","name":"ComfortFit Polo Shirt","description":"Cotton blend polo shirt in multiple colors","category":"Clothing","unit_price":39.99}'
    '{"sku":"CLOTH-PANTS-001","name":"FlexWear Cargo Pants","description":"Durable cargo pants with stretch fabric","category":"Clothing","unit_price":59.99}'
    '{"sku":"CLOTH-BOOTS-001","name":"TrailMaster Boots","description":"Steel-toe waterproof work boots","category":"Clothing","unit_price":149.99}'
    '{"sku":"FOOD-COFFEE-001","name":"Mountain Blend Coffee","description":"Premium Arabica coffee beans 1kg bag","category":"Food & Beverage","unit_price":24.99}'
    '{"sku":"FOOD-TEA-001","name":"Zen Garden Green Tea","description":"Organic Japanese green tea 100 sachets","category":"Food & Beverage","unit_price":18.99}'
    '{"sku":"FOOD-WATER-001","name":"PureSpring Water Pack","description":"24-pack natural spring water 500ml bottles","category":"Food & Beverage","unit_price":12.99}'
    '{"sku":"FOOD-SNACK-001","name":"NutriBar Variety Pack","description":"Mixed protein bars 12-pack assorted flavors","category":"Food & Beverage","unit_price":29.99}'
    '{"sku":"TOOL-DRILL-001","name":"PowerDrive Cordless Drill","description":"18V lithium-ion cordless drill with 2 batteries","category":"Tools","unit_price":89.99}'
    '{"sku":"TOOL-SAW-001","name":"PrecisionCut Circular Saw","description":"185mm circular saw with laser guide","category":"Tools","unit_price":119.99}'
    '{"sku":"TOOL-KIT-001","name":"ProMaster Toolkit 150pc","description":"150-piece professional hand tool set in carry case","category":"Tools","unit_price":199.99}'
    '{"sku":"TOOL-MEASURE-001","name":"LaserPoint Distance Meter","description":"Digital laser distance meter up to 50m","category":"Tools","unit_price":69.99}'
  )

  for product in "${products[@]}"; do
    local sku
    sku=$(echo "$product" | jq -r '.sku')
    local response
    response=$(api_post "/api/v1/products" "$product" "$ADMIN_TOKEN")
    local status
    status=$(get_status "$response")
    if [ "$status" = "201" ] || [ "$status" = "200" ]; then
      local product_id
      product_id=$(json_field "$response" '.data.id')
      PRODUCT_IDS+=("$product_id")
      log_info "Created product: $sku — ID: ${product_id:0:8}..."
    elif [ "$status" = "409" ] || [ "$status" = "400" ]; then
      log_warn "Product $sku already exists, fetching ID..."
      # Fetch existing product by listing with SKU filter
      local list_response
      list_response=$(api_get "/api/v1/products?sku=${sku}&limit=1" "$ADMIN_TOKEN")
      local existing_id
      existing_id=$(json_field "$list_response" '.data[0].id // empty')
      if [ -n "$existing_id" ] && [ "$existing_id" != "null" ]; then
        PRODUCT_IDS+=("$existing_id")
        log_info "  Found existing: ${existing_id:0:8}..."
      else
        log_warn "  Could not fetch existing product ID for $sku"
      fi
    else
      log_error "Failed to create product $sku (HTTP $status)"
      echo "$response" | sed '$d'
    fi
  done

  log_info "Total products tracked: ${#PRODUCT_IDS[@]}"
}

# ---------- 3. Warehouses ----------

declare -a WAREHOUSE_IDS=()

create_warehouses() {
  log_step "Creating Warehouses (3 warehouses)"

  local warehouses=(
    '{"name":"Kyiv Central Warehouse","address":"Khreshchatyk St, 22, Kyiv, Ukraine 01001"}'
    '{"name":"Lviv Distribution Center","address":"Svobody Ave, 45, Lviv, Ukraine 79000"}'
    '{"name":"Odesa Regional Hub","address":"Derybasivska St, 12, Odesa, Ukraine 65000"}'
  )

  local names=("Kyiv Central Warehouse" "Lviv Distribution Center" "Odesa Regional Hub")

  for i in "${!warehouses[@]}"; do
    local name="${names[$i]}"
    local response
    response=$(api_post "/api/v1/warehouses" "${warehouses[$i]}" "$ADMIN_TOKEN")
    local status
    status=$(get_status "$response")
    if [ "$status" = "201" ] || [ "$status" = "200" ]; then
      local wh_id
      wh_id=$(json_field "$response" '.data.id')
      WAREHOUSE_IDS+=("$wh_id")
      log_info "Created warehouse: $name — ID: ${wh_id:0:8}..."
    elif [ "$status" = "409" ] || [ "$status" = "400" ]; then
      log_warn "Warehouse '$name' may already exist, fetching..."
      local list_response
      list_response=$(api_get "/api/v1/warehouses?name=$(echo "$name" | head -c 10 | sed 's/ /%20/g')&limit=1" "$ADMIN_TOKEN")
      local existing_id
      existing_id=$(json_field "$list_response" '.data[0].id // empty')
      if [ -n "$existing_id" ] && [ "$existing_id" != "null" ]; then
        WAREHOUSE_IDS+=("$existing_id")
        log_info "  Found existing: ${existing_id:0:8}..."
      fi
    else
      log_error "Failed to create warehouse '$name' (HTTP $status)"
      echo "$response" | sed '$d'
    fi
  done

  log_info "Total warehouses tracked: ${#WAREHOUSE_IDS[@]}"
}

# ---------- 4. Stock (inbound adjustments) ----------

setup_stock() {
  log_step "Setting up Stock (inbound adjustments for all products at all warehouses)"

  if [ ${#PRODUCT_IDS[@]} -eq 0 ] || [ ${#WAREHOUSE_IDS[@]} -eq 0 ]; then
    log_error "No products or warehouses — cannot set up stock"
    return 1
  fi

  # Quantities per product (varied for realism)
  local quantities=(200 150 100 300 80 60 120 90 250 400 350 180 500 450 600 300 160 140 110 220)
  local thresholds=(30 25 15 50 10 8 20 15 40 60 50 25 80 70 100 50 25 20 15 35)

  # Check if stock is already populated (idempotency)
  local existing_stock
  existing_stock=$(api_get "/api/v1/stock?limit=1" "$ADMIN_TOKEN")
  local existing_count
  existing_count=$(json_field "$existing_stock" '.meta.total // 0')
  if [ "$existing_count" -gt 0 ]; then
    log_warn "Stock already populated ($existing_count entries), skipping adjustments"
    log_info "  (To re-seed stock, clear the stock table first)"
    # Still set thresholds (idempotent PUT)
    for wi in "${!WAREHOUSE_IDS[@]}"; do
      local wh_id="${WAREHOUSE_IDS[$wi]}"
      for pi in "${!PRODUCT_IDS[@]}"; do
        local prod_id="${PRODUCT_IDS[$pi]}"
        local threshold="${thresholds[$pi]}"
        api_put "/api/v1/stock/threshold" \
          "{\"product_id\":\"${prod_id}\",\"warehouse_id\":\"${wh_id}\",\"threshold\":${threshold}}" \
          "$ADMIN_TOKEN" > /dev/null 2>&1
      done
    done
    return 0
  fi

  for wi in "${!WAREHOUSE_IDS[@]}"; do
    local wh_id="${WAREHOUSE_IDS[$wi]}"
    for pi in "${!PRODUCT_IDS[@]}"; do
      local prod_id="${PRODUCT_IDS[$pi]}"
      # Vary quantity by warehouse (60%, 100%, 80% of base)
      local base_qty="${quantities[$pi]}"
      local multipliers=(60 100 80)
      local qty=$(( base_qty * multipliers[wi] / 100 ))

      local response
      response=$(api_post "/api/v1/stock/adjust" \
        "{\"product_id\":\"${prod_id}\",\"warehouse_id\":\"${wh_id}\",\"quantity\":${qty},\"type\":\"inbound\",\"reference\":\"seed-initial-stock\"}" \
        "$ADMIN_TOKEN")
      local status
      status=$(get_status "$response")
      if [ "$status" = "200" ] || [ "$status" = "201" ]; then
        : # success, silent
      else
        log_warn "Stock adjust failed for product ${prod_id:0:8} at warehouse ${wh_id:0:8} (HTTP $status)"
      fi

      # Set min threshold
      local threshold="${thresholds[$pi]}"
      api_put "/api/v1/stock/threshold" \
        "{\"product_id\":\"${prod_id}\",\"warehouse_id\":\"${wh_id}\",\"threshold\":${threshold}}" \
        "$ADMIN_TOKEN" > /dev/null 2>&1
    done
    log_info "Stocked warehouse ${wh_id:0:8}... with ${#PRODUCT_IDS[@]} products"
  done
}

# ---------- 5. Carriers ----------

declare -a CARRIER_IDS=()

create_carriers() {
  log_step "Creating Carriers (3 carriers: ground, air, sea)"

  local carriers=(
    '{"name":"TransContinental Express","type":"ground","cost_per_km":2.50}'
    '{"name":"SkyFreight International","type":"air","cost_per_km":8.75}'
    '{"name":"OceanLine Cargo","type":"sea","cost_per_km":1.20}'
  )

  local names=("TransContinental Express" "SkyFreight International" "OceanLine Cargo")

  for i in "${!carriers[@]}"; do
    local name="${names[$i]}"
    local response
    response=$(api_post "/api/v1/carriers" "${carriers[$i]}" "$ADMIN_TOKEN")
    local status
    status=$(get_status "$response")
    if [ "$status" = "201" ] || [ "$status" = "200" ]; then
      local carrier_id
      carrier_id=$(json_field "$response" '.data.id')
      CARRIER_IDS+=("$carrier_id")
      log_info "Created carrier: $name — ID: ${carrier_id:0:8}..."
    elif [ "$status" = "409" ] || [ "$status" = "400" ]; then
      log_warn "Carrier '$name' may already exist, fetching..."
      local list_response
      list_response=$(api_get "/api/v1/carriers?limit=10" "$ADMIN_TOKEN")
      local existing_id
      existing_id=$(json_field "$list_response" ".data[] | select(.name==\"$name\") | .id" 2>/dev/null || echo "")
      if [ -n "$existing_id" ] && [ "$existing_id" != "null" ]; then
        CARRIER_IDS+=("$existing_id")
        log_info "  Found existing: ${existing_id:0:8}..."
      fi
    else
      log_error "Failed to create carrier '$name' (HTTP $status)"
      echo "$response" | sed '$d'
    fi
  done

  log_info "Total carriers tracked: ${#CARRIER_IDS[@]}"
}

# ---------- 6. Orders ----------

declare -a ORDER_IDS=()

create_orders() {
  log_step "Creating Orders (10 orders in various statuses)"

  if [ ${#PRODUCT_IDS[@]} -lt 4 ]; then
    log_error "Not enough products to create orders"
    return 1
  fi

  # Use operator token for creating orders
  local token="${USER_TOKENS[operator]:-$ADMIN_TOKEN}"

  # Idempotency: check if orders already exist
  local existing_orders
  existing_orders=$(api_get "/api/v1/orders?limit=1" "$token")
  local existing_count
  existing_count=$(json_field "$existing_orders" '.meta.total // 0')
  if [ "$existing_count" -ge 10 ]; then
    log_warn "Orders already seeded ($existing_count orders found), skipping"
    # Populate ORDER_IDS from existing data for shipment creation
    local all_orders
    all_orders=$(api_get "/api/v1/orders?limit=10&sort=created_at&order=asc" "$token")
    for idx in $(seq 0 9); do
      local oid
      oid=$(json_field "$all_orders" ".data[$idx].id // empty")
      ORDER_IDS+=("${oid:-}")
    done
    return 0
  fi

  # Order 1: Pending — fresh order
  local order1="{\"customer_name\":\"Dmytro Ivanenko\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[0]}\",\"name\":\"ProBook Laptop 15\",\"quantity\":2,\"unit_price\":899.99},
    {\"product_id\":\"${PRODUCT_IDS[3]}\",\"name\":\"NoiseFree Pro Headphones\",\"quantity\":3,\"unit_price\":199.99}
  ]}"

  # Order 2: Pending — another fresh order
  local order2="{\"customer_name\":\"Nataliia Fedorenko\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[8]}\",\"name\":\"AllWeather Jacket\",\"quantity\":5,\"unit_price\":129.99},
    {\"product_id\":\"${PRODUCT_IDS[11]}\",\"name\":\"TrailMaster Boots\",\"quantity\":5,\"unit_price\":149.99}
  ]}"

  # Order 3: will be Confirmed
  local order3="{\"customer_name\":\"Serhii Popovskyi\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[4]}\",\"name\":\"ErgoMax Office Chair\",\"quantity\":10,\"unit_price\":349.99},
    {\"product_id\":\"${PRODUCT_IDS[5]}\",\"name\":\"StandUp Adjustable Desk\",\"quantity\":10,\"unit_price\":549.99}
  ]}"

  # Order 4: will be Processing
  local order4="{\"customer_name\":\"Anna Mykhailenko\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[12]}\",\"name\":\"Mountain Blend Coffee\",\"quantity\":50,\"unit_price\":24.99},
    {\"product_id\":\"${PRODUCT_IDS[13]}\",\"name\":\"Zen Garden Green Tea\",\"quantity\":30,\"unit_price\":18.99},
    {\"product_id\":\"${PRODUCT_IDS[15]}\",\"name\":\"NutriBar Variety Pack\",\"quantity\":20,\"unit_price\":29.99}
  ]}"

  # Order 5: will be Shipped
  local order5="{\"customer_name\":\"Viktor Kozachenko\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[16]}\",\"name\":\"PowerDrive Cordless Drill\",\"quantity\":15,\"unit_price\":89.99},
    {\"product_id\":\"${PRODUCT_IDS[18]}\",\"name\":\"ProMaster Toolkit 150pc\",\"quantity\":10,\"unit_price\":199.99}
  ]}"

  # Order 6: will be Delivered
  local order6="{\"customer_name\":\"Olha Lebedynska\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[1]}\",\"name\":\"SmartPhone X12\",\"quantity\":3,\"unit_price\":699.99},
    {\"product_id\":\"${PRODUCT_IDS[2]}\",\"name\":\"TabPro 10\",\"quantity\":2,\"unit_price\":449.99}
  ]}"

  # Order 7: will be Completed
  local order7="{\"customer_name\":\"Pavlo Orlovskyi\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[6]}\",\"name\":\"BookWall Shelf Unit\",\"quantity\":4,\"unit_price\":179.99},
    {\"product_id\":\"${PRODUCT_IDS[7]}\",\"name\":\"SecureLock Filing Cabinet\",\"quantity\":6,\"unit_price\":229.99}
  ]}"

  # Order 8: will be Cancelled
  local order8="{\"customer_name\":\"Kateryna Novak\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[9]}\",\"name\":\"ComfortFit Polo Shirt\",\"quantity\":100,\"unit_price\":39.99},
    {\"product_id\":\"${PRODUCT_IDS[10]}\",\"name\":\"FlexWear Cargo Pants\",\"quantity\":50,\"unit_price\":59.99}
  ]}"

  # Order 9: Pending — large order
  local order9="{\"customer_name\":\"Andrii Smiian\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[14]}\",\"name\":\"PureSpring Water Pack\",\"quantity\":200,\"unit_price\":12.99},
    {\"product_id\":\"${PRODUCT_IDS[15]}\",\"name\":\"NutriBar Variety Pack\",\"quantity\":100,\"unit_price\":29.99}
  ]}"

  # Order 10: will be Confirmed
  local order10="{\"customer_name\":\"Yuliia Morozenko\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[17]}\",\"name\":\"PrecisionCut Circular Saw\",\"quantity\":8,\"unit_price\":119.99},
    {\"product_id\":\"${PRODUCT_IDS[19]}\",\"name\":\"LaserPoint Distance Meter\",\"quantity\":12,\"unit_price\":69.99}
  ]}"

  local orders=("$order1" "$order2" "$order3" "$order4" "$order5" "$order6" "$order7" "$order8" "$order9" "$order10")
  local customers=("Dmytro Ivanenko" "Nataliia Fedorenko" "Serhii Popovskyi" "Anna Mykhailenko" "Viktor Kozachenko" "Olha Lebedynska" "Pavlo Orlovskyi" "Kateryna Novak" "Andrii Smiian" "Yuliia Morozenko")

  # Target statuses for each order
  # 1:pending, 2:pending, 3:confirmed, 4:processing, 5:shipped,
  # 6:delivered, 7:completed, 8:cancelled, 9:pending, 10:confirmed

  for i in "${!orders[@]}"; do
    local response
    response=$(api_post "/api/v1/orders" "${orders[$i]}" "$token")
    local status
    status=$(get_status "$response")
    if [ "$status" = "201" ] || [ "$status" = "200" ]; then
      local order_id
      order_id=$(json_field "$response" '.data.id')
      ORDER_IDS+=("$order_id")
      log_info "Created order #$((i+1)): ${customers[$i]} — ID: ${order_id:0:8}..."
    else
      log_warn "Failed to create order for ${customers[$i]} (HTTP $status)"
      ORDER_IDS+=("")
    fi
  done

  # Now transition orders to target statuses
  log_info "Transitioning orders to target statuses..."

  # Order 3 (index 2): pending → confirmed
  if [ -n "${ORDER_IDS[2]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[2]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    log_info "  Order #3 → confirmed"
  fi

  # Order 4 (index 3): pending → confirmed → processing
  if [ -n "${ORDER_IDS[3]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[3]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[3]}/status" '{"status":"processing"}' "$token" > /dev/null 2>&1
    log_info "  Order #4 → processing"
  fi

  # Order 5 (index 4): pending → confirmed → processing → shipped
  if [ -n "${ORDER_IDS[4]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[4]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[4]}/status" '{"status":"processing"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[4]}/status" '{"status":"shipped"}' "$token" > /dev/null 2>&1
    log_info "  Order #5 → shipped"
  fi

  # Order 6 (index 5): pending → confirmed → processing → shipped → delivered
  if [ -n "${ORDER_IDS[5]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[5]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[5]}/status" '{"status":"processing"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[5]}/status" '{"status":"shipped"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[5]}/status" '{"status":"delivered"}' "$token" > /dev/null 2>&1
    log_info "  Order #6 → delivered"
  fi

  # Order 7 (index 6): full workflow → completed
  if [ -n "${ORDER_IDS[6]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[6]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[6]}/status" '{"status":"processing"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[6]}/status" '{"status":"shipped"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[6]}/status" '{"status":"delivered"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[6]}/status" '{"status":"completed"}' "$token" > /dev/null 2>&1
    log_info "  Order #7 → completed"
  fi

  # Order 8 (index 7): pending → cancelled
  if [ -n "${ORDER_IDS[7]:-}" ]; then
    api_post "/api/v1/orders/${ORDER_IDS[7]}/cancel" '{"reason":"Customer changed their mind — requested full refund"}' "$token" > /dev/null 2>&1
    log_info "  Order #8 → cancelled"
  fi

  # Order 10 (index 9): pending → confirmed
  if [ -n "${ORDER_IDS[9]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[9]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    log_info "  Order #10 → confirmed"
  fi
}

# ---------- 7. Shipments ----------

declare -a SHIPMENT_IDS=()

create_shipments() {
  log_step "Creating Shipments (5 shipments with various statuses)"

  if [ ${#ORDER_IDS[@]} -lt 8 ] || [ ${#CARRIER_IDS[@]} -lt 3 ] || [ ${#WAREHOUSE_IDS[@]} -lt 1 ]; then
    log_error "Not enough orders, carriers, or warehouses for shipments"
    return 1
  fi

  local token="${USER_TOKENS[logistics_manager]:-$ADMIN_TOKEN}"

  # Idempotency: check if shipments already exist
  local existing_shipments
  existing_shipments=$(api_get "/api/v1/shipments?limit=1" "$ADMIN_TOKEN")
  local existing_count
  existing_count=$(json_field "$existing_shipments" '.meta.total // 0')
  if [ "$existing_count" -ge 5 ]; then
    log_warn "Shipments already seeded ($existing_count found), skipping"
    return 0
  fi

  local shipments=(
    "{\"order_id\":\"${ORDER_IDS[4]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[0]}\",\"carrier_id\":\"${CARRIER_IDS[0]}\",\"address\":\"Soborna St, 15, Kharkiv, Ukraine 61000\"}"
    "{\"order_id\":\"${ORDER_IDS[5]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[1]}\",\"carrier_id\":\"${CARRIER_IDS[1]}\",\"address\":\"Sobornyi Ave, 32, Dnipro, Ukraine 49000\"}"
    "{\"order_id\":\"${ORDER_IDS[6]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[0]}\",\"carrier_id\":\"${CARRIER_IDS[0]}\",\"address\":\"Pushkinska St, 88, Vinnytsia, Ukraine 21000\"}"
    "{\"order_id\":\"${ORDER_IDS[3]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[2]}\",\"carrier_id\":\"${CARRIER_IDS[2]}\",\"address\":\"Yavornytskoho Ave, 5, Zaporizhzhia, Ukraine 69000\"}"
    "{\"order_id\":\"${ORDER_IDS[2]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[1]}\",\"carrier_id\":\"${CARRIER_IDS[0]}\",\"address\":\"Soborna St, 21, Poltava, Ukraine 36000\"}"
  )

  local descriptions=("Shipped order for Viktor Kozachenko" "Delivered order for Olha Lebedynska" "Completed order for Pavlo Orlovskyi" "Processing order for Anna Mykhailenko" "Confirmed order for Serhii Popovskyi")

  for i in "${!shipments[@]}"; do
    local response
    response=$(api_post "/api/v1/shipments" "${shipments[$i]}" "$ADMIN_TOKEN")
    local status
    status=$(get_status "$response")
    if [ "$status" = "201" ] || [ "$status" = "200" ]; then
      local shipment_id
      shipment_id=$(json_field "$response" '.data.id')
      SHIPMENT_IDS+=("$shipment_id")
      log_info "Created shipment #$((i+1)): ${descriptions[$i]} — ID: ${shipment_id:0:8}..."
    else
      log_warn "Failed to create shipment #$((i+1)) (HTTP $status)"
      SHIPMENT_IDS+=("")
    fi
  done

  # Transition shipments to various statuses
  log_info "Transitioning shipments to target statuses..."

  # Shipment 1 (index 0): created → picked_up → in_transit (shipped)
  if [ -n "${SHIPMENT_IDS[0]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[0]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[0]}/status" '{"status":"in_transit"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #1 → in_transit"
  fi

  # Shipment 2 (index 1): created → picked_up → in_transit → delivered
  if [ -n "${SHIPMENT_IDS[1]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[1]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[1]}/status" '{"status":"in_transit"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[1]}/status" '{"status":"delivered"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #2 → delivered"
  fi

  # Shipment 3 (index 2): created → picked_up → in_transit → delivered
  if [ -n "${SHIPMENT_IDS[2]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[2]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[2]}/status" '{"status":"in_transit"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[2]}/status" '{"status":"delivered"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #3 → delivered"
  fi

  # Shipment 4 (index 3): created → picked_up (in progress)
  if [ -n "${SHIPMENT_IDS[3]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[3]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #4 → picked_up"
  fi

  # Shipment 5 (index 4): stays as created
  if [ -n "${SHIPMENT_IDS[4]:-}" ]; then
    log_info "  Shipment #5 → created (no transition)"
  fi
}

# ---------- 8. Notifications (per-staff distribution) ----------

seed_notifications() {
  log_step "Seeding Notifications across Staff Users"

  if ! docker compose exec -T postgres psql -U postgres -d chainorchestra >/dev/null 2>&1 <<'PSQL'
SELECT 1;
PSQL
  then
    log_warn "Cannot reach postgres via docker compose; skipping notification seed"
    return 0
  fi

  docker compose exec -T postgres psql -U postgres -d chainorchestra -v ON_ERROR_STOP=1 <<'PSQL'
DELETE FROM notifications.notification
WHERE created_at >= NOW() - INTERVAL '120 days';

WITH staff AS (
  SELECT id::text AS user_id, role
  FROM users.users
  WHERE deleted_at IS NULL
    AND role IN ('admin','warehouse_manager','logistics_manager','analyst','operator')
),
templates AS (
  SELECT * FROM (VALUES
    ('admin',             'system',            'System backup completed',     'Database backup completed at 03:14 UTC.'),
    ('admin',             'system',            'New user registered',         'A new operator account joined the system.'),
    ('admin',             'order_updated',     'Order escalation',            'Order ORD-#### requires admin attention.'),
    ('warehouse_manager', 'low_stock',         'Low stock alert',             'Floor Mats stock dropped below threshold at Kyiv Hub.'),
    ('warehouse_manager', 'low_stock',         'Low stock alert',             'LED Bulbs are running low at Lviv DC.'),
    ('warehouse_manager', 'stock_changed',     'Stock replenished',           'Office Chairs were restocked at Odesa Depot (+120 units).'),
    ('warehouse_manager', 'stock_changed',     'Stock adjustment',            'Stock recount adjusted Steel Bolts at Kyiv Hub.'),
    ('logistics_manager', 'shipment_created',  'Shipment created',            'Shipment for order ORD-#### dispatched with Nova Express.'),
    ('logistics_manager', 'shipment_updated',  'Shipment in transit',         'Shipment SH-#### is now in transit through Vinnytsia.'),
    ('logistics_manager', 'shipment_updated',  'Shipment delivered',          'Shipment SH-#### was delivered successfully in Kharkiv.'),
    ('logistics_manager', 'shipment_updated',  'Carrier delay reported',      'Carrier reported a 6h delay for shipment SH-####.'),
    ('analyst',           'system',            'Weekly report ready',         'Sales analytics for the past week is now available.'),
    ('analyst',           'system',            'Anomaly detected',            'Unusual sales spike detected for category Electronics.'),
    ('analyst',           'system',            'Forecast refresh complete',   'Demand forecast model refreshed for the next 30 days.'),
    ('operator',          'order_created',     'New order received',          'Order ORD-#### was placed by a new customer.'),
    ('operator',          'order_created',     'New order received',          'Order ORD-#### needs operator review.'),
    ('operator',          'order_cancelled',   'Order cancelled',             'Order ORD-#### was cancelled by the customer.'),
    ('operator',          'order_updated',     'Order status updated',        'Order ORD-#### moved to processing.')
  ) AS t(role, type, title, message)
)
INSERT INTO notifications.notification (id, user_id, type, title, message, status, read_at, created_at)
SELECT
  gen_random_uuid(),
  s.user_id,
  t.type,
  t.title,
  REPLACE(t.message, '####', LPAD((10000 + (random() * 89999)::int)::text, 5, '0')),
  CASE WHEN random() < 0.55 THEN 'pending' ELSE 'read' END AS status,
  CASE WHEN random() < 0.55 THEN NULL ELSE NOW() - (random() * interval '40 days') END AS read_at,
  NOW() - (random() * interval '60 days') AS created_at
FROM staff s
JOIN templates t ON t.role = s.role
CROSS JOIN generate_series(1, 8) g;

UPDATE notifications.notification SET status = 'pending' WHERE status = 'unread';

SELECT u.role,
       u.email,
       COUNT(n.id) FILTER (WHERE n.status = 'pending') AS unread,
       COUNT(n.id)                                    AS total
FROM users.users u
LEFT JOIN notifications.notification n ON n.user_id = u.id::text
WHERE u.deleted_at IS NULL
  AND u.role IN ('admin','warehouse_manager','logistics_manager','analyst','operator')
GROUP BY u.role, u.email
ORDER BY unread DESC;
PSQL

  log_info "Notification seed complete"
}

# ---------- 8b. Quick-cancellation forensic scenarios ----------

seed_quick_cancellations() {
  log_step "Seeding Quick-Cancellation Forensic Scenarios"

  if ! docker compose exec -T postgres psql -U postgres -d chainorchestra >/dev/null 2>&1 <<'PSQL'
SELECT 1;
PSQL
  then
    log_warn "Cannot reach postgres via docker compose; skipping quick-cancel seed"
    return 0
  fi

  docker compose exec -T postgres psql -U postgres -d chainorchestra -v ON_ERROR_STOP=1 <<'PSQL'
WITH transcontinental AS (
  SELECT id FROM logistics.carrier WHERE name = 'TransContinental Express' LIMIT 1
),
oceanline AS (
  SELECT id FROM logistics.carrier WHERE name = 'OceanLine Cargo' LIMIT 1
),
candidate_shipments AS (
  SELECT s.id AS shipment_id, s.order_id, s.created_at, s.address, s.carrier_id
  FROM logistics.shipment s
  JOIN orders.orders o ON o.id::text = s.order_id
  WHERE s.deleted_at IS NULL
    AND o.deleted_at IS NULL
    AND o.status IN ('shipped','delivered','completed')
    AND s.created_at >= NOW() - INTERVAL '90 days'
  ORDER BY random()
  LIMIT 50
),
hotspot_pool AS (
  SELECT shipment_id, order_id, created_at,
         'Soborna St, ' || (10 + (random()*89)::int)::text || ', Kharkiv, Ukraine 61000' AS address,
         (SELECT id FROM transcontinental) AS carrier_id
  FROM candidate_shipments
  ORDER BY random() LIMIT 12
),
secondary_pool AS (
  SELECT shipment_id, order_id, created_at,
         'Sobornyi Ave, ' || (10 + (random()*89)::int)::text || ', Dnipro, Ukraine 49000' AS address,
         (SELECT id FROM oceanline) AS carrier_id
  FROM candidate_shipments
  WHERE shipment_id NOT IN (SELECT shipment_id FROM hotspot_pool)
  ORDER BY random() LIMIT 5
),
spread_pool AS (
  SELECT shipment_id, order_id, created_at, address, carrier_id
  FROM candidate_shipments
  WHERE shipment_id NOT IN (SELECT shipment_id FROM hotspot_pool UNION ALL SELECT shipment_id FROM secondary_pool)
  ORDER BY random() LIMIT 8
),
all_targets AS (
  SELECT * FROM hotspot_pool
  UNION ALL SELECT * FROM secondary_pool
  UNION ALL SELECT * FROM spread_pool
)
UPDATE logistics.shipment s
SET address = t.address,
    carrier_id = t.carrier_id,
    status = 'cancelled',
    updated_at = t.created_at + make_interval(mins => 5 + (random()*50)::int)
FROM all_targets t
WHERE s.id = t.shipment_id;

UPDATE orders.orders o
SET status = 'cancelled',
    cancel_reason = CASE
      WHEN c.name = 'TransContinental Express' AND s.address LIKE '%Kharkiv%'
        THEN 'Customer reported address mismatch — investigate carrier handover'
      WHEN c.name = 'OceanLine Cargo' AND s.address LIKE '%Dnipro%'
        THEN 'Damaged on dispatch — pickup refused'
      ELSE 'Customer cancelled after dispatch'
    END,
    updated_at = s.created_at + make_interval(mins => 5 + (random()*50)::int)
FROM logistics.shipment s
JOIN logistics.carrier c ON c.id = s.carrier_id
WHERE o.id::text = s.order_id
  AND s.status = 'cancelled'
  AND s.created_at >= NOW() - INTERVAL '90 days';
PSQL

  log_info "Quick-cancel forensic seed complete (~25 events)"
}

# ---------- 8c. Postal tracking events + recipient/sender backfill ----------

seed_postal_tracking() {
  log_step "Seeding Postal Tracking Events"

  if ! docker compose exec -T postgres psql -U postgres -d chainorchestra >/dev/null 2>&1 <<'PSQL'
SELECT 1;
PSQL
  then
    log_warn "Cannot reach postgres via docker compose; skipping postal seed"
    return 0
  fi

  docker compose exec -T postgres psql -U postgres -d chainorchestra -v ON_ERROR_STOP=1 <<'PSQL'
WITH order_customer AS (
  SELECT id::text AS oid, customer_name FROM orders.orders
),
parsed AS (
  SELECT
    s.id, s.order_id, s.address,
    oc.customer_name,
    trim(split_part(s.address, ',', 1)) AS street,
    trim(split_part(s.address, ',', 2)) AS street_number,
    trim(split_part(s.address, ',', 3)) AS city,
    trim(split_part(s.address, ',', 4)) AS country_zip
  FROM logistics.shipment s
  LEFT JOIN order_customer oc ON oc.oid = s.order_id
  WHERE s.recipient = '{}'::jsonb OR s.recipient IS NULL
)
UPDATE logistics.shipment s
SET recipient = jsonb_build_object(
    'full_name', COALESCE(p.customer_name, 'Customer'),
    'phone',     '+38050' || LPAD((FLOOR(RANDOM()*9999999))::text, 7, '0'),
    'email',     LOWER(REPLACE(COALESCE(p.customer_name, 'customer'), ' ', '.')) || '@example.com',
    'street',    NULLIF(p.street || ', ' || p.street_number, ', '),
    'city',      NULLIF(p.city, ''),
    'country',   CASE WHEN p.country_zip LIKE '%Ukraine%' THEN 'Ukraine' ELSE TRIM(p.country_zip) END,
    'postcode',  TRIM(SUBSTRING(p.country_zip FROM '\d+'))
)
FROM parsed p
WHERE s.id = p.id;

UPDATE logistics.shipment s
SET sender = jsonb_build_object(
    'company',   w.name,
    'full_name', 'Warehouse Operations',
    'phone',     '+38067' || LPAD((FLOOR(RANDOM()*9999999))::text, 7, '0'),
    'email',     'ops+' || REPLACE(LOWER(w.name), ' ', '') || '@chainorchestra.local',
    'street',    w.address,
    'city',      CASE
        WHEN w.name ILIKE '%Kyiv%'  THEN 'Kyiv'
        WHEN w.name ILIKE '%Lviv%'  THEN 'Lviv'
        WHEN w.name ILIKE '%Odesa%' THEN 'Odesa'
        ELSE 'Kyiv' END,
    'country',   'Ukraine'
)
FROM inventory.warehouse w
WHERE s.warehouse_id = w.id::text AND (s.sender = '{}'::jsonb OR s.sender IS NULL);

UPDATE logistics.shipment
SET estimated_delivery_at = created_at + (96 + (random() * 72)::int) * INTERVAL '1 hour'
WHERE estimated_delivery_at IS NULL;

DELETE FROM logistics.shipment_event;
DELETE FROM logistics.delivery_attempt;
UPDATE logistics.shipment SET delivery_attempts = 0;

WITH events_to_add AS (
  SELECT id, created_at, updated_at, status,
         sender->>'city' AS origin_city,
         recipient->>'city' AS dest_city
  FROM logistics.shipment
)
INSERT INTO logistics.shipment_event (shipment_id, event_type, location_city, location_hub, notes, occurred_at, recorded_by)
SELECT id, 'label_created', origin_city, NULL, 'Shipping label created', created_at, 'system'
FROM events_to_add
UNION ALL
SELECT id, 'picked_up', origin_city, NULL, 'Picked up by carrier from warehouse',
       created_at + INTERVAL '2 hours', 'carrier_api'
FROM events_to_add WHERE status NOT IN ('cancelled', 'created', 'label_created', 'awaiting_pickup')
UNION ALL
SELECT id, 'in_transit', NULL, NULL, 'In transit between hubs',
       created_at + INTERVAL '8 hours', 'carrier_api'
FROM events_to_add WHERE status NOT IN ('cancelled', 'created', 'label_created', 'awaiting_pickup', 'picked_up')
UNION ALL
SELECT id, 'hub_arrived', dest_city, dest_city || ' Sorting Hub',
       'Arrived at destination sorting hub',
       created_at + INTERVAL '24 hours', 'carrier_api'
FROM events_to_add WHERE status IN ('out_for_delivery', 'delivered', 'delivery_attempted', 'failed', 'returned', 'returned_to_sender', 'held_at_office', 'in_transit', 'at_hub')
UNION ALL
SELECT id, 'out_for_delivery', dest_city, NULL, 'Out for delivery with driver',
       created_at + INTERVAL '48 hours', 'carrier_api'
FROM events_to_add WHERE status IN ('out_for_delivery', 'delivered', 'delivery_attempted', 'failed', 'returned', 'returned_to_sender', 'held_at_office')
UNION ALL
SELECT id, 'delivered', dest_city, NULL, 'Package delivered', updated_at, 'driver'
FROM events_to_add WHERE status = 'delivered'
UNION ALL
SELECT id, 'delivery_attempted', dest_city, NULL, 'No one home — left notice', updated_at, 'driver'
FROM events_to_add WHERE status IN ('delivery_attempted', 'failed', 'returned', 'returned_to_sender')
UNION ALL
SELECT id, 'returned_to_sender', origin_city, NULL,
       'Returned to sender after failed attempts',
       updated_at + INTERVAL '24 hours', 'system'
FROM events_to_add WHERE status IN ('returned', 'returned_to_sender');

UPDATE logistics.shipment
SET delivered_at = updated_at,
    delivery_signature = recipient->>'full_name',
    delivery_attempts = 1
WHERE status = 'delivered';

INSERT INTO logistics.delivery_attempt (shipment_id, attempt_number, reason, notes, occurred_at)
SELECT id, 1, 'no_one_home', 'First attempt — left notice', updated_at - INTERVAL '24 hours'
FROM logistics.shipment WHERE status IN ('delivery_attempted', 'failed', 'returned', 'returned_to_sender');

INSERT INTO logistics.delivery_attempt (shipment_id, attempt_number, reason, notes, occurred_at)
SELECT id, 2, 'no_one_home', 'Second attempt', updated_at - INTERVAL '12 hours'
FROM logistics.shipment WHERE status IN ('returned', 'returned_to_sender');

INSERT INTO logistics.delivery_attempt (shipment_id, attempt_number, reason, notes, occurred_at)
SELECT id, 3, 'refused', 'Final attempt — unreachable', updated_at - INTERVAL '4 hours'
FROM logistics.shipment WHERE status IN ('returned', 'returned_to_sender');

UPDATE logistics.shipment SET delivery_attempts = 1
WHERE status IN ('delivery_attempted', 'failed');
UPDATE logistics.shipment SET delivery_attempts = 3
WHERE status IN ('returned', 'returned_to_sender');

UPDATE logistics.shipment
SET current_location_city = recipient->>'city',
    current_location_hub  = (recipient->>'city') || ' Sorting Hub'
WHERE status IN ('in_transit', 'at_hub', 'out_for_delivery');
PSQL

  log_info "Postal tracking seed complete"
}

# ---------- 9. Verify ----------

verify_data() {
  log_step "Verifying Seed Data"

  local token="$ADMIN_TOKEN"

  # Check users
  local users_response
  users_response=$(api_get "/api/v1/users?limit=10" "$token")
  local user_count
  user_count=$(json_field "$users_response" '.meta.total // 0')
  log_info "Users: $user_count (expected: 5)"

  # Check products
  local products_response
  products_response=$(api_get "/api/v1/products?limit=1" "$token")
  local product_count
  product_count=$(json_field "$products_response" '.meta.total // 0')
  log_info "Products: $product_count (expected: 20)"

  # Check warehouses
  local warehouses_response
  warehouses_response=$(api_get "/api/v1/warehouses?limit=1" "$token")
  local warehouse_count
  warehouse_count=$(json_field "$warehouses_response" '.meta.total // 0')
  log_info "Warehouses: $warehouse_count (expected: 3)"

  # Check orders
  local orders_response
  orders_response=$(api_get "/api/v1/orders?limit=1" "$token")
  local order_count
  order_count=$(json_field "$orders_response" '.meta.total // 0')
  log_info "Orders: $order_count (expected: 10)"

  # Check order stats
  local stats_response
  stats_response=$(api_get "/api/v1/orders/stats" "$token")
  local stats_body
  stats_body=$(echo "$stats_response" | sed '$d')
  log_info "Order stats: $stats_body"

  # Check carriers
  local carriers_response
  carriers_response=$(api_get "/api/v1/carriers?limit=1" "$token")
  local carrier_count
  carrier_count=$(json_field "$carriers_response" '.meta.total // 0')
  log_info "Carriers: $carrier_count (expected: 3)"

  # Check shipments
  local shipments_response
  shipments_response=$(api_get "/api/v1/shipments?limit=1" "$token")
  local shipment_count
  shipment_count=$(json_field "$shipments_response" '.meta.total // 0')
  log_info "Shipments: $shipment_count (expected: 5+)"

  # Check stock
  local stock_response
  stock_response=$(api_get "/api/v1/stock?limit=1" "$token")
  local stock_count
  stock_count=$(json_field "$stock_response" '.meta.total // 0')
  log_info "Stock entries: $stock_count (expected: 60)"

  # Check low stock
  local low_stock_response
  low_stock_response=$(api_get "/api/v1/stock/low" "$token")
  local low_stock_body
  low_stock_body=$(echo "$low_stock_response" | sed '$d')
  local low_count
  low_count=$(echo "$low_stock_body" | jq '.data | length' 2>/dev/null || echo "0")
  log_info "Low-stock items: $low_count"
}

# ---------- Main ----------

main() {
  echo -e "${BLUE}"
  echo "  ╔══════════════════════════════════════╗"
  echo "  ║   ChainOrchestra — Seed Data Script  ║"
  echo "  ╚══════════════════════════════════════╝"
  echo -e "${NC}"

  # Check for jq
  if ! command -v jq &> /dev/null; then
    log_error "jq is required but not installed. Install with: brew install jq"
    exit 1
  fi

  wait_for_gateway
  admin_login
  create_users
  create_products
  create_warehouses
  setup_stock
  create_carriers
  create_orders
  create_shipments
  seed_notifications
  seed_quick_cancellations
  seed_postal_tracking
  verify_data

  log_step "Seed Complete"
  log_info "All seed data has been populated successfully!"
  log_info ""
  log_info "Test credentials (see /Users/haradrim/Desktop/test-credentials.txt for the strong-password set):"
  log_info "  Admin:             admin@chainorchestra.local"
  log_info "  Operator:          ivan.petrenko@chainorchestra.local"
  log_info "  Warehouse Manager: maria.kovalenko@chainorchestra.local"
  log_info "  Logistics Manager: oleksii.shevchenko@chainorchestra.local"
  log_info "  Analyst:           olena.bondarenko@chainorchestra.local"
}

main "$@"
