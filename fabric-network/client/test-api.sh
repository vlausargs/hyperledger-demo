#!/bin/bash
# Test script for Fabric Client API

BASE_URL="http://localhost:8080/api/v1"

echo "=== Testing Fabric Client API ==="
echo ""

# Health check
echo "1. Health Check:"
curl -s "$BASE_URL/../health" | jq '.'
echo ""

# Get all assets
echo "2. Get All Assets:"
curl -s "$BASE_URL/assets" | jq '.'
echo ""

# Create a new asset
echo "3. Create New Asset:"
curl -X POST "$BASE_URL/assets" \
  -H "Content-Type: application/json" \
  -d '{"ID":"api_test_1","color":"purple","size":20,"owner":"API_User","appraisedValue":500}' | jq '.'
echo ""

# Get specific asset
echo "4. Get Specific Asset:"
curl -s "$BASE_URL/assets/api_test_1" | jq '.'
echo ""

# Update asset
echo "5. Update Asset:"
curl -X PUT "$BASE_URL/assets/api_test_1" \
  -H "Content-Type: application/json" \
  -d '{"color":"orange","size":25,"owner":"API_User2","appraisedValue":600}' | jq '.'
echo ""

# Transfer asset
echo "6. Transfer Asset:"
curl -X POST "$BASE_URL/assets/api_test_1/transfer" \
  -H "Content-Type: application/json" \
  -d '{"newOwner":"API_User3"}' | jq '.'
echo ""

# Get asset history
echo "7. Get Asset History:"
curl -s "$BASE_URL/assets/api_test_1/history" | jq '.'
echo ""

echo "=== API Testing Complete ==="
