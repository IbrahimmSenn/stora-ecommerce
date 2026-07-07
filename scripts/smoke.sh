#!/usr/bin/env bash
# Post-deploy validation: service health, database connectivity, and the
# critical user flows (browse, search, login, add to cart). Exits non-zero on
# the first failure so deploy.sh / the pipeline can halt or roll back.
#
# Usage: smoke.sh [base-url]   (default http://localhost:8080)
set -euo pipefail

BASE="${1:-http://localhost:8080}"
CURL="curl -fsS --max-time 15"

fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }
pass() { echo "  ok: $*"; }

echo "smoke tests against $BASE"

# 1. Readiness — proves the API is up AND can reach Postgres + RabbitMQ.
$CURL "$BASE/ready" | grep -q '"status":"ok"' \
  || fail "/ready reports degraded or unreachable"
pass "service healthy, database and broker reachable"

# 2. Product catalog (browse flow, exercises a real DB read).
products=$($CURL "$BASE/api/v1/products?page=1")
echo "$products" | grep -q '"products":\[{' || fail "product listing empty or malformed"
pass "product listing returns items"

# 3. Product search.
$CURL "$BASE/api/v1/products?q=a&page=1" | grep -q '"total"' || fail "product search failed"
pass "product search responds"

# 4. Login with the seeded demo customer.
token=$($CURL -X POST "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"customer@shop.com","password":"customer123"}' \
  | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
[ -n "$token" ] || fail "login did not return an access token"
pass "demo customer login"

# 5. Add the first in-stock product to the cart.
product_id=$(echo "$products" | sed -n 's/.*"id":"\([0-9a-f-]\{36\}\)".*/\1/p' | head -1)
[ -n "$product_id" ] || fail "could not extract a product id"
$CURL -X POST "$BASE/api/v1/cart/items" \
  -H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$product_id\",\"quantity\":1}" \
  | grep -q '"items"' || fail "add to cart failed"
pass "add to cart"

# 6. SEO surface.
$CURL "$BASE/sitemap.xml" | grep -q '<urlset' || fail "sitemap.xml missing"
pass "sitemap served"

echo "smoke tests passed"
