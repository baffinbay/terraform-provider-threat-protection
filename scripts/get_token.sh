#!/bin/bash

# Source the .env file to get CLIENT_ID and CLIENT_SECRET
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo "Error: .env file not found."
    exit 1
fi

if [ -z "$BAFFINBAY_CLIENT_ID" ] || [ -z "$BAFFINBAY_CLIENT_SECRET" ]; then
    echo "Error: BAFFINBAY_CLIENT_ID and BAFFINBAY_CLIENT_SECRET must be set in .env"
    exit 1
fi

echo "Retrieving access token..."

RESPONSE=$(curl -s -X POST https://m2m-auth.baffinbay.com/oauth/token \
    -H "Content-Type: application/json" \
    -d "{
    \"client_id\": \"$BAFFINBAY_CLIENT_ID\",
    \"client_secret\": \"$BAFFINBAY_CLIENT_SECRET\",
    \"grant_type\": \"client_credentials\",
    \"audience\": \"https://portal.baffinbay.com\"
}")

# Use python or jq if available for reliable JSON parsing, falling back to grep/sed
if command -v jq &> /dev/null; then
    ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r .access_token)
else
    # Simple grep/sed extraction for standard JSON response
    ACCESS_TOKEN=$(echo "$RESPONSE" | grep -o '"access_token":"[^"]*"' | sed 's/"access_token":"//;s/"//')
fi

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" == "null" ]; then
    echo "Error: Failed to retrieve access token."
    echo "Response: $RESPONSE"
    exit 1
fi

echo "Access token retrieved successfully."

# Update .env file safely
# Create a temp file with all lines EXCEPT BAFFINBAY_API_KEY
grep -v "^BAFFINBAY_API_KEY=" .env > .env.tmp

# Append the new key
echo "BAFFINBAY_API_KEY=$ACCESS_TOKEN" >> .env.tmp

# Replace the original file
mv .env.tmp .env

echo "BAFFINBAY_API_KEY has been updated in .env"