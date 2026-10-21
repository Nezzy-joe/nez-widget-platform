#!/usr/bin/env bash
set -Eeuo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:8080}"
SEED_TENANT_NAME="${SEED_TENANT_NAME:-Capstone Demo}"

command -v curl >/dev/null || { echo "curl is required."; exit 1; }
command -v jq >/dev/null || { echo "jq is required."; exit 1; }

if [[ -z "${SEED_EMAIL:-}" ]]; then
  read -r -p "Demo account email: " SEED_EMAIL
fi

if [[ -z "${SEED_PASSWORD:-}" ]]; then
  read -r -s -p "Demo account password (8–72 characters): " SEED_PASSWORD
  printf '\n'
fi

if [[ -z "$SEED_EMAIL" || ${#SEED_PASSWORD} -lt 8 || ${#SEED_PASSWORD} -gt 72 ]]; then
  echo "Provide an email and a password between 8 and 72 characters."
  exit 1
fi

api_call() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local token="${4:-}"
  local raw

  if [[ -n "$body" ]]; then
    raw="$(printf '%s' "$body" | curl -sS \
      -H 'Content-Type: application/json' \
      ${token:+-H "Authorization: Bearer $token"} \
      -X "$method" --data-binary @- \
      -w $'\n%{http_code}' "${API_BASE_URL}${path}")"
  else
    raw="$(curl -sS \
      -H 'Content-Type: application/json' \
      ${token:+-H "Authorization: Bearer $token"} \
      -X "$method" \
      -w $'\n%{http_code}' "${API_BASE_URL}${path}")"
  fi

  HTTP_STATUS="${raw##*$'\n'}"
  HTTP_BODY="${raw%$'\n'*}"
}

login_body="$(jq -n --arg email "$SEED_EMAIL" --arg password "$SEED_PASSWORD" \
  '{email:$email,password:$password}')"

api_call POST /auth/login "$login_body"

if [[ "$HTTP_STATUS" == "200" ]]; then
  TOKEN="$(jq -er '.access_token' <<<"$HTTP_BODY")"
elif [[ "$HTTP_STATUS" == "401" ]]; then
  register_body="$(jq -n \
    --arg tenant "$SEED_TENANT_NAME" \
    --arg email "$SEED_EMAIL" \
    --arg password "$SEED_PASSWORD" \
    '{tenant_name:$tenant,email:$email,password:$password}')"

  api_call POST /auth/register "$register_body"

  if [[ "$HTTP_STATUS" == "201" ]]; then
    api_call POST /auth/login "$login_body"
    [[ "$HTTP_STATUS" == "200" ]] || {
      echo "Account registered, but login failed. Check the API logs."
      exit 1
    }
    TOKEN="$(jq -er '.access_token' <<<"$HTTP_BODY")"
  elif [[ "$HTTP_STATUS" == "409" ]]; then
    echo "That email already exists, but login failed. Check the password."
    exit 1
  else
    echo "Registration failed with HTTP $HTTP_STATUS."
    jq -r '.error // "No error details returned."' <<<"$HTTP_BODY"
    exit 1
  fi
else
  echo "Login failed with HTTP $HTTP_STATUS."
  jq -r '.error // "No error details returned."' <<<"$HTTP_BODY"
  exit 1
fi

api_call GET /widgets "" "$TOKEN"

if [[ "$HTTP_STATUS" != "200" ]]; then
  echo "Could not list widgets (HTTP $HTTP_STATUS)."
  exit 1
fi

PUBLIC_ID="$(jq -r --arg title 'Capstone Demo Widget' \
  '.widgets[]? | select(.title == $title) | .public_id' \
  <<<"$HTTP_BODY" | head -n 1)"

if [[ -z "$PUBLIC_ID" ]]; then
  widget_body="$(jq -n '{
    type:"contact",
    title:"Capstone Demo Widget",
    description:"A demonstration lead-capture form.",
    button_text:"Send message",
    display_options:{},
    fields:[
      {name:"name",label:"Full name",field_type:"text",required:true},
      {name:"email",label:"Email address",field_type:"email",required:true},
      {name:"message",label:"Message",field_type:"textarea",required:true}
    ]
  }')"

  api_call POST /widgets "$widget_body" "$TOKEN"

  if [[ "$HTTP_STATUS" != "201" ]]; then
    echo "Widget creation failed with HTTP $HTTP_STATUS."
    jq -r '.error // "No error details returned."' <<<"$HTTP_BODY"
    exit 1
  fi

  PUBLIC_ID="$(jq -er '.widget.public_id' <<<"$HTTP_BODY")"
  echo "Created Capstone Demo Widget."
else
  echo "Capstone Demo Widget already exists; no duplicate created."
fi

printf 'Public widget ID: %s\n' "$PUBLIC_ID"
printf '\nEmbed snippet for local testing:\n'
printf '<script src="%s/widget/v1/widget.js?widget_id=%s" defer></script>\n' "$API_BASE_URL" "$PUBLIC_ID"
