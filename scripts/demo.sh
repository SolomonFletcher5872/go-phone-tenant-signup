#!/bin/sh
set -eu

base_url=${SERVICE_URL:-http://localhost:8080}
phone=${PHONE_NUMBER:?set PHONE_NUMBER in E.164 format}

curl --fail-with-body -X POST "$base_url/signup/code" \
  -H 'Content-Type: application/json' \
  -d "{\"tenant_name\":\"Northwind Tools\",\"phone\":\"$phone\",\"locale\":\"en\"}"

printf 'Code: '
read -r code
curl --fail-with-body -X POST "$base_url/signup/verify" \
  -H 'Content-Type: application/json' \
  -d "{\"phone\":\"$phone\",\"code\":\"$code\"}"
