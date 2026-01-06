#!/bin/bash
set -e

# Sercha Core Quickstart Script
# This script walks you through setting up Sercha Core with a GitHub connector.
#
# Prerequisites:
#   - Docker Compose running (docker compose up -d)
#   - GitHub OAuth App created with callback URL: http://localhost:8080/api/v1/oauth/callback
#
# Usage:
#   ./quickstart.sh
#
# Or set environment variables first:
#   export GITHUB_CLIENT_ID="your-client-id"
#   export GITHUB_CLIENT_SECRET="your-client-secret"
#   ./quickstart.sh

API_URL="${API_URL:-http://localhost:8080}"

echo "=== Sercha Core Quickstart ==="
echo ""

# Check if services are running
echo "Checking if Sercha is running..."
if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
    echo "Error: Sercha is not running at $API_URL"
    echo "Start it with: docker compose up -d"
    exit 1
fi
echo "Sercha is running."
echo ""

# Get admin credentials
if [ -z "$ADMIN_EMAIL" ]; then
    read -p "Admin email [admin@example.com]: " ADMIN_EMAIL
    ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
fi

if [ -z "$ADMIN_PASSWORD" ]; then
    read -s -p "Admin password [changeme]: " ADMIN_PASSWORD
    echo ""
    ADMIN_PASSWORD="${ADMIN_PASSWORD:-changeme}"
fi

if [ -z "$ADMIN_NAME" ]; then
    ADMIN_NAME="Admin"
fi

# Step 1: Create admin user
echo ""
echo "=== Step 1: Creating admin user ==="
SETUP_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/setup" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\", \"name\": \"$ADMIN_NAME\"}" \
    2>&1) || true

if echo "$SETUP_RESPONSE" | grep -q "already"; then
    echo "Admin user already exists, continuing..."
else
    echo "Admin user created."
fi

# Step 2: Login
echo ""
echo "=== Step 2: Logging in ==="
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "Error: Failed to login. Response: $LOGIN_RESPONSE"
    exit 1
fi
echo "Logged in successfully."

# Step 3: Initialize Vespa
echo ""
echo "=== Step 3: Initializing Vespa schema ==="
VESPA_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/admin/vespa/connect" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"dev_mode": true}')

if echo "$VESPA_RESPONSE" | grep -q "error"; then
    echo "Vespa may already be initialized or there was an issue: $VESPA_RESPONSE"
else
    echo "Vespa schema initialized."
fi

# Step 4: Configure GitHub provider
echo ""
echo "=== Step 4: Configuring GitHub OAuth ==="

if [ -z "$GITHUB_CLIENT_ID" ]; then
    read -p "GitHub OAuth Client ID: " GITHUB_CLIENT_ID
fi

if [ -z "$GITHUB_CLIENT_SECRET" ]; then
    read -s -p "GitHub OAuth Client Secret: " GITHUB_CLIENT_SECRET
    echo ""
fi

PROVIDER_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/providers/github/config" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"client_id\": \"$GITHUB_CLIENT_ID\",
        \"client_secret\": \"$GITHUB_CLIENT_SECRET\",
        \"redirect_uri\": \"$API_URL/api/v1/oauth/callback\",
        \"enabled\": true
    }")

echo "GitHub provider configured."

# Step 5: Start OAuth flow
echo ""
echo "=== Step 5: Starting GitHub OAuth flow ==="
OAUTH_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/oauth/github/authorize" \
    -H "Authorization: Bearer $TOKEN")

AUTH_URL=$(echo "$OAUTH_RESPONSE" | grep -o '"authorization_url":"[^"]*"' | cut -d'"' -f4 | sed 's/\\u0026/\&/g')

if [ -z "$AUTH_URL" ]; then
    echo "Error: Failed to get OAuth URL. Response: $OAUTH_RESPONSE"
    exit 1
fi

echo ""
echo "Open this URL in your browser to authorize:"
echo ""
echo "  $AUTH_URL"
echo ""

# Try to open browser automatically
if command -v open &> /dev/null; then
    read -p "Open in browser? [Y/n]: " OPEN_BROWSER
    if [ "$OPEN_BROWSER" != "n" ] && [ "$OPEN_BROWSER" != "N" ]; then
        open "$AUTH_URL"
    fi
elif command -v xdg-open &> /dev/null; then
    read -p "Open in browser? [Y/n]: " OPEN_BROWSER
    if [ "$OPEN_BROWSER" != "n" ] && [ "$OPEN_BROWSER" != "N" ]; then
        xdg-open "$AUTH_URL"
    fi
fi

echo ""
read -p "Press Enter after you've authorized the app in your browser..."

# Step 6: List installations
echo ""
echo "=== Step 6: Fetching installations ==="
INSTALLATIONS_RESPONSE=$(curl -s "$API_URL/api/v1/installations" \
    -H "Authorization: Bearer $TOKEN")

echo "$INSTALLATIONS_RESPONSE"

INSTALLATION_ID=$(echo "$INSTALLATIONS_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$INSTALLATION_ID" ]; then
    echo "Error: No installations found. Did you complete the OAuth flow?"
    exit 1
fi

echo ""
echo "Using installation: $INSTALLATION_ID"

# Step 7: List containers (repos)
echo ""
echo "=== Step 7: Fetching available repositories ==="
CONTAINERS_RESPONSE=$(curl -s "$API_URL/api/v1/installations/$INSTALLATION_ID/containers" \
    -H "Authorization: Bearer $TOKEN")

echo "$CONTAINERS_RESPONSE"

# Extract first container ID
CONTAINER_ID=$(echo "$CONTAINERS_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$CONTAINER_ID" ]; then
    echo "Error: No repositories found."
    exit 1
fi

echo ""
read -p "Repository to index [$CONTAINER_ID]: " SELECTED_REPO
SELECTED_REPO="${SELECTED_REPO:-$CONTAINER_ID}"

# Step 8: Create source
echo ""
echo "=== Step 8: Creating source ==="
SOURCE_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/sources" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"$SELECTED_REPO\",
        \"provider_type\": \"github\",
        \"installation_id\": \"$INSTALLATION_ID\",
        \"selected_containers\": [\"$SELECTED_REPO\"]
    }")

echo "$SOURCE_RESPONSE"

SOURCE_ID=$(echo "$SOURCE_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$SOURCE_ID" ]; then
    echo "Error: Failed to create source."
    exit 1
fi

echo ""
echo "Source created: $SOURCE_ID"

# Step 9: Trigger sync
echo ""
echo "=== Step 9: Triggering sync ==="
SYNC_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/sources/$SOURCE_ID/sync" \
    -H "Authorization: Bearer $TOKEN")

echo "$SYNC_RESPONSE"
echo ""
echo "Sync started. Waiting for completion..."

# Poll for sync completion
for i in {1..30}; do
    sleep 2
    STATUS_RESPONSE=$(curl -s "$API_URL/api/v1/sources/$SOURCE_ID" \
        -H "Authorization: Bearer $TOKEN")

    SYNC_STATUS=$(echo "$STATUS_RESPONSE" | grep -o '"sync_status":"[^"]*"' | cut -d'"' -f4)

    if [ "$SYNC_STATUS" = "completed" ] || [ "$SYNC_STATUS" = "idle" ]; then
        echo "Sync completed!"
        break
    elif [ "$SYNC_STATUS" = "failed" ]; then
        echo "Sync failed."
        echo "$STATUS_RESPONSE"
        exit 1
    else
        echo "Sync status: $SYNC_STATUS"
    fi
done

# Step 10: Search
echo ""
echo "=== Step 10: Search ==="
read -p "Enter search query [README]: " SEARCH_QUERY
SEARCH_QUERY="${SEARCH_QUERY:-README}"

SEARCH_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/search" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"$SEARCH_QUERY\"}")

echo ""
echo "$SEARCH_RESPONSE"

echo ""
echo "=== Quickstart Complete! ==="
echo ""
echo "Your Sercha instance is now set up with:"
echo "  - Admin user: $ADMIN_EMAIL"
echo "  - GitHub installation: $INSTALLATION_ID"
echo "  - Source: $SOURCE_ID ($SELECTED_REPO)"
echo ""
echo "API URL: $API_URL"
echo "Token: $TOKEN"
echo ""
echo "Try searching:"
echo "  curl -X POST $API_URL/api/v1/search \\"
echo "    -H \"Authorization: Bearer \$TOKEN\" \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -d '{\"query\": \"your search\"}'"
