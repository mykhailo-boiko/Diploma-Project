#!/usr/bin/env bash
set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@chainorchestra.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "\n${BLUE}=== $1 ===${NC}"; }


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

json_field() {
  echo "$1" | sed '$d' | jq -r "$2"
}


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


declare -A USER_TOKENS
declare -A USER_IDS

create_users() {
  log_step "Creating Users (5 users, one per role)"

  local users=(
    '{"email":"ivan.petrov@chainorchestra.local","password":"Operator1!","first_name":"Ivan","last_name":"Petrov","role":"operator"}'
    '{"email":"maria.kuznetsova@chainorchestra.local","password":"Warehouse1!","first_name":"Maria","last_name":"Kuznetsova","role":"warehouse_manager"}'
    '{"email":"alexei.volkov@chainorchestra.local","password":"Logistics1!","first_name":"Alexei","last_name":"Volkov","role":"logistics_manager"}'
    '{"email":"elena.sokolova@chainorchestra.local","password":"Analyst1!","first_name":"Elena","last_name":"Sokolova","role":"analyst"}'
  )

  local emails=(
    "ivan.petrov@chainorchestra.local"
    "maria.kuznetsova@chainorchestra.local"
    "alexei.volkov@chainorchestra.local"
    "elena.sokolova@chainorchestra.local"
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

  log_info "Logging in as each user to get tokens..."

  local all_emails=("$ADMIN_EMAIL" "${emails[@]}")
  local all_passwords=("$ADMIN_PASSWORD" "Operator1!" "Warehouse1!" "Logistics1!" "Analyst1!")
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


declare -a WAREHOUSE_IDS=()

create_warehouses() {
  log_step "Creating Warehouses (3 warehouses)"

  local warehouses=(
    '{"name":"Moscow Central Warehouse","address":"123 Tverskaya St, Moscow, Russia 125009"}'
    '{"name":"Saint Petersburg Distribution Center","address":"45 Nevsky Prospekt, Saint Petersburg, Russia 191186"}'
    '{"name":"Novosibirsk Regional Hub","address":"78 Krasny Prospekt, Novosibirsk, Russia 630099"}'
  )

  local names=("Moscow Central Warehouse" "Saint Petersburg Distribution Center" "Novosibirsk Regional Hub")

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


setup_stock() {
  log_step "Setting up Stock (inbound adjustments for all products at all warehouses)"

  if [ ${
    log_error "No products or warehouses — cannot set up stock"
    return 1
  fi

  local quantities=(200 150 100 300 80 60 120 90 250 400 350 180 500 450 600 300 160 140 110 220)
  local thresholds=(30 25 15 50 10 8 20 15 40 60 50 25 80 70 100 50 25 20 15 35)

  local existing_stock
  existing_stock=$(api_get "/api/v1/stock?limit=1" "$ADMIN_TOKEN")
  local existing_count
  existing_count=$(json_field "$existing_stock" '.meta.total // 0')
  if [ "$existing_count" -gt 0 ]; then
    log_warn "Stock already populated ($existing_count entries), skipping adjustments"
    log_info "  (To re-seed stock, clear the stock table first)"
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
        :
      else
        log_warn "Stock adjust failed for product ${prod_id:0:8} at warehouse ${wh_id:0:8} (HTTP $status)"
      fi

      local threshold="${thresholds[$pi]}"
      api_put "/api/v1/stock/threshold" \
        "{\"product_id\":\"${prod_id}\",\"warehouse_id\":\"${wh_id}\",\"threshold\":${threshold}}" \
        "$ADMIN_TOKEN" > /dev/null 2>&1
    done
    log_info "Stocked warehouse ${wh_id:0:8}... with ${#PRODUCT_IDS[@]} products"
  done
}


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


declare -a ORDER_IDS=()

create_orders() {
  log_step "Creating Orders (10 orders in various statuses)"

  if [ ${
    log_error "Not enough products to create orders"
    return 1
  fi

  local token="${USER_TOKENS[operator]:-$ADMIN_TOKEN}"

  local existing_orders
  existing_orders=$(api_get "/api/v1/orders?limit=1" "$token")
  local existing_count
  existing_count=$(json_field "$existing_orders" '.meta.total // 0')
  if [ "$existing_count" -ge 10 ]; then
    log_warn "Orders already seeded ($existing_count orders found), skipping"
    local all_orders
    all_orders=$(api_get "/api/v1/orders?limit=10&sort=created_at&order=asc" "$token")
    for idx in $(seq 0 9); do
      local oid
      oid=$(json_field "$all_orders" ".data[$idx].id // empty")
      ORDER_IDS+=("${oid:-}")
    done
    return 0
  fi

  local order1="{\"customer_name\":\"Dmitry Ivanov\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[0]}\",\"name\":\"ProBook Laptop 15\",\"quantity\":2,\"unit_price\":899.99},
    {\"product_id\":\"${PRODUCT_IDS[3]}\",\"name\":\"NoiseFree Pro Headphones\",\"quantity\":3,\"unit_price\":199.99}
  ]}"

  local order2="{\"customer_name\":\"Natalia Fedorova\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[8]}\",\"name\":\"AllWeather Jacket\",\"quantity\":5,\"unit_price\":129.99},
    {\"product_id\":\"${PRODUCT_IDS[11]}\",\"name\":\"TrailMaster Boots\",\"quantity\":5,\"unit_price\":149.99}
  ]}"

  local order3="{\"customer_name\":\"Sergei Popov\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[4]}\",\"name\":\"ErgoMax Office Chair\",\"quantity\":10,\"unit_price\":349.99},
    {\"product_id\":\"${PRODUCT_IDS[5]}\",\"name\":\"StandUp Adjustable Desk\",\"quantity\":10,\"unit_price\":549.99}
  ]}"

  local order4="{\"customer_name\":\"Anna Mikhailova\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[12]}\",\"name\":\"Mountain Blend Coffee\",\"quantity\":50,\"unit_price\":24.99},
    {\"product_id\":\"${PRODUCT_IDS[13]}\",\"name\":\"Zen Garden Green Tea\",\"quantity\":30,\"unit_price\":18.99},
    {\"product_id\":\"${PRODUCT_IDS[15]}\",\"name\":\"NutriBar Variety Pack\",\"quantity\":20,\"unit_price\":29.99}
  ]}"

  local order5="{\"customer_name\":\"Viktor Kozlov\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[16]}\",\"name\":\"PowerDrive Cordless Drill\",\"quantity\":15,\"unit_price\":89.99},
    {\"product_id\":\"${PRODUCT_IDS[18]}\",\"name\":\"ProMaster Toolkit 150pc\",\"quantity\":10,\"unit_price\":199.99}
  ]}"

  local order6="{\"customer_name\":\"Olga Lebedeva\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[1]}\",\"name\":\"SmartPhone X12\",\"quantity\":3,\"unit_price\":699.99},
    {\"product_id\":\"${PRODUCT_IDS[2]}\",\"name\":\"TabPro 10\",\"quantity\":2,\"unit_price\":449.99}
  ]}"

  local order7="{\"customer_name\":\"Pavel Orlov\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[6]}\",\"name\":\"BookWall Shelf Unit\",\"quantity\":4,\"unit_price\":179.99},
    {\"product_id\":\"${PRODUCT_IDS[7]}\",\"name\":\"SecureLock Filing Cabinet\",\"quantity\":6,\"unit_price\":229.99}
  ]}"

  local order8="{\"customer_name\":\"Ekaterina Novikova\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[9]}\",\"name\":\"ComfortFit Polo Shirt\",\"quantity\":100,\"unit_price\":39.99},
    {\"product_id\":\"${PRODUCT_IDS[10]}\",\"name\":\"FlexWear Cargo Pants\",\"quantity\":50,\"unit_price\":59.99}
  ]}"

  local order9="{\"customer_name\":\"Andrei Smirnov\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[14]}\",\"name\":\"PureSpring Water Pack\",\"quantity\":200,\"unit_price\":12.99},
    {\"product_id\":\"${PRODUCT_IDS[15]}\",\"name\":\"NutriBar Variety Pack\",\"quantity\":100,\"unit_price\":29.99}
  ]}"

  local order10="{\"customer_name\":\"Yulia Morozova\",\"items\":[
    {\"product_id\":\"${PRODUCT_IDS[17]}\",\"name\":\"PrecisionCut Circular Saw\",\"quantity\":8,\"unit_price\":119.99},
    {\"product_id\":\"${PRODUCT_IDS[19]}\",\"name\":\"LaserPoint Distance Meter\",\"quantity\":12,\"unit_price\":69.99}
  ]}"

  local orders=("$order1" "$order2" "$order3" "$order4" "$order5" "$order6" "$order7" "$order8" "$order9" "$order10")
  local customers=("Dmitry Ivanov" "Natalia Fedorova" "Sergei Popov" "Anna Mikhailova" "Viktor Kozlov" "Olga Lebedeva" "Pavel Orlov" "Ekaterina Novikova" "Andrei Smirnov" "Yulia Morozova")


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

  log_info "Transitioning orders to target statuses..."

  if [ -n "${ORDER_IDS[2]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[2]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    log_info "  Order #3 → confirmed"
  fi

  if [ -n "${ORDER_IDS[3]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[3]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[3]}/status" '{"status":"processing"}' "$token" > /dev/null 2>&1
    log_info "  Order #4 → processing"
  fi

  if [ -n "${ORDER_IDS[4]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[4]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[4]}/status" '{"status":"processing"}' "$token" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/orders/${ORDER_IDS[4]}/status" '{"status":"shipped"}' "$token" > /dev/null 2>&1
    log_info "  Order #5 → shipped"
  fi

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

  if [ -n "${ORDER_IDS[7]:-}" ]; then
    api_post "/api/v1/orders/${ORDER_IDS[7]}/cancel" '{"reason":"Customer changed their mind — requested full refund"}' "$token" > /dev/null 2>&1
    log_info "  Order #8 → cancelled"
  fi

  if [ -n "${ORDER_IDS[9]:-}" ]; then
    api_put "/api/v1/orders/${ORDER_IDS[9]}/status" '{"status":"confirmed"}' "$token" > /dev/null 2>&1
    log_info "  Order #10 → confirmed"
  fi
}


declare -a SHIPMENT_IDS=()

create_shipments() {
  log_step "Creating Shipments (5 shipments with various statuses)"

  if [ ${
    log_error "Not enough orders, carriers, or warehouses for shipments"
    return 1
  fi

  local token="${USER_TOKENS[logistics_manager]:-$ADMIN_TOKEN}"

  local existing_shipments
  existing_shipments=$(api_get "/api/v1/shipments?limit=1" "$ADMIN_TOKEN")
  local existing_count
  existing_count=$(json_field "$existing_shipments" '.meta.total // 0')
  if [ "$existing_count" -ge 5 ]; then
    log_warn "Shipments already seeded ($existing_count found), skipping"
    return 0
  fi

  local shipments=(
    "{\"order_id\":\"${ORDER_IDS[4]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[0]}\",\"carrier_id\":\"${CARRIER_IDS[0]}\",\"address\":\"15 Lenina St, Kazan, Russia 420111\"}"
    "{\"order_id\":\"${ORDER_IDS[5]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[1]}\",\"carrier_id\":\"${CARRIER_IDS[1]}\",\"address\":\"32 Gagarina Blvd, Yekaterinburg, Russia 620075\"}"
    "{\"order_id\":\"${ORDER_IDS[6]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[0]}\",\"carrier_id\":\"${CARRIER_IDS[0]}\",\"address\":\"88 Pushkina St, Sochi, Russia 354000\"}"
    "{\"order_id\":\"${ORDER_IDS[3]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[2]}\",\"carrier_id\":\"${CARRIER_IDS[2]}\",\"address\":\"5 Mira Prospekt, Vladivostok, Russia 690091\"}"
    "{\"order_id\":\"${ORDER_IDS[2]}\",\"warehouse_id\":\"${WAREHOUSE_IDS[1]}\",\"carrier_id\":\"${CARRIER_IDS[0]}\",\"address\":\"21 Arbat St, Moscow, Russia 119002\"}"
  )

  local descriptions=("Shipped order for Viktor Kozlov" "Delivered order for Olga Lebedeva" "Completed order for Pavel Orlov" "Processing order for Anna Mikhailova" "Confirmed order for Sergei Popov")

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

  log_info "Transitioning shipments to target statuses..."

  if [ -n "${SHIPMENT_IDS[0]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[0]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[0]}/status" '{"status":"in_transit"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #1 → in_transit"
  fi

  if [ -n "${SHIPMENT_IDS[1]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[1]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[1]}/status" '{"status":"in_transit"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[1]}/status" '{"status":"delivered"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #2 → delivered"
  fi

  if [ -n "${SHIPMENT_IDS[2]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[2]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[2]}/status" '{"status":"in_transit"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    sleep 0.2
    api_put "/api/v1/shipments/${SHIPMENT_IDS[2]}/status" '{"status":"delivered"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #3 → delivered"
  fi

  if [ -n "${SHIPMENT_IDS[3]:-}" ]; then
    api_put "/api/v1/shipments/${SHIPMENT_IDS[3]}/status" '{"status":"picked_up"}' "$ADMIN_TOKEN" > /dev/null 2>&1
    log_info "  Shipment #4 → picked_up"
  fi

  if [ -n "${SHIPMENT_IDS[4]:-}" ]; then
    log_info "  Shipment #5 → created (no transition)"
  fi
}


verify_data() {
  log_step "Verifying Seed Data"

  local token="$ADMIN_TOKEN"

  local users_response
  users_response=$(api_get "/api/v1/users?limit=10" "$token")
  local user_count
  user_count=$(json_field "$users_response" '.meta.total // 0')
  log_info "Users: $user_count (expected: 5)"

  local products_response
  products_response=$(api_get "/api/v1/products?limit=1" "$token")
  local product_count
  product_count=$(json_field "$products_response" '.meta.total // 0')
  log_info "Products: $product_count (expected: 20)"

  local warehouses_response
  warehouses_response=$(api_get "/api/v1/warehouses?limit=1" "$token")
  local warehouse_count
  warehouse_count=$(json_field "$warehouses_response" '.meta.total // 0')
  log_info "Warehouses: $warehouse_count (expected: 3)"

  local orders_response
  orders_response=$(api_get "/api/v1/orders?limit=1" "$token")
  local order_count
  order_count=$(json_field "$orders_response" '.meta.total // 0')
  log_info "Orders: $order_count (expected: 10)"

  local stats_response
  stats_response=$(api_get "/api/v1/orders/stats" "$token")
  local stats_body
  stats_body=$(echo "$stats_response" | sed '$d')
  log_info "Order stats: $stats_body"

  local carriers_response
  carriers_response=$(api_get "/api/v1/carriers?limit=1" "$token")
  local carrier_count
  carrier_count=$(json_field "$carriers_response" '.meta.total // 0')
  log_info "Carriers: $carrier_count (expected: 3)"

  local shipments_response
  shipments_response=$(api_get "/api/v1/shipments?limit=1" "$token")
  local shipment_count
  shipment_count=$(json_field "$shipments_response" '.meta.total // 0')
  log_info "Shipments: $shipment_count (expected: 5+)"

  local stock_response
  stock_response=$(api_get "/api/v1/stock?limit=1" "$token")
  local stock_count
  stock_count=$(json_field "$stock_response" '.meta.total // 0')
  log_info "Stock entries: $stock_count (expected: 60)"

  local low_stock_response
  low_stock_response=$(api_get "/api/v1/stock/low" "$token")
  local low_stock_body
  low_stock_body=$(echo "$low_stock_response" | sed '$d')
  local low_count
  low_count=$(echo "$low_stock_body" | jq '.data | length' 2>/dev/null || echo "0")
  log_info "Low-stock items: $low_count"
}


main() {
  echo -e "${BLUE}"
  echo "  ╔══════════════════════════════════════╗"
  echo "  ║   ChainOrchestra — Seed Data Script  ║"
  echo "  ╚══════════════════════════════════════╝"
  echo -e "${NC}"

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
  verify_data

  log_step "Seed Complete"
  log_info "All seed data has been populated successfully!"
  log_info ""
  log_info "Test credentials:"
  log_info "  Admin:             admin@chainorchestra.local / admin123"
  log_info "  Operator:          ivan.petrov@chainorchestra.local / Operator1!"
  log_info "  Warehouse Manager: maria.kuznetsova@chainorchestra.local / Warehouse1!"
  log_info "  Logistics Manager: alexei.volkov@chainorchestra.local / Logistics1!"
  log_info "  Analyst:           elena.sokolova@chainorchestra.local / Analyst1!"
}

main "$@"
