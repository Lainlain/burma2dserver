#!/bin/bash

# ThaiMaster2D Lottery API Test Script

echo "========================================"
echo "🎰 ThaiMaster2D Lottery Server Test"
echo "========================================"
echo ""

# Test 1: Health check
echo "1️⃣  Testing health check endpoint..."
curl -s http://localhost:8080/ | jq .
echo ""
echo ""

# Test current data
echo "📊 Testing GET /api/burma2d/live..."
curl -s http://localhost:4545/api/burma2d/live | jq .

# Test update endpoint
echo ""
echo "📮 Testing POST /api/burma2d/update..."
curl -X POST http://localhost:4545/api/burma2d/update \
  -H "Content-Type: application/json" \
  -d '{
    "live": "22",
    "status": "On",
    "set1200": "15",
    "value1200": "89",
    "result1200": "589",
    "set430": "67",
    "value430": "34",
    "result430": "134",
    "modern930": "845",
    "internet930": "921",
    "modern200": "376",
    "internet200": "542",
    "updatetime": "12:01:45 16/10/2025"
  }' | jq .
echo ""
echo ""

# Test 4: Get updated lottery data
echo "4️⃣  Getting updated lottery data..."
curl -s http://localhost:4545/api/burma2d/live | jq .
echo ""
echo ""

echo "========================================"
echo "✅ All tests completed!"
echo "========================================"
echo ""
echo "📡 To test SSE streaming, open another terminal and run:"
echo "   curl -N http://localhost:4545/api/burma2d/stream"
echo ""
echo "📮 To send updates, use:"
echo "   curl -X POST http://localhost:4545/api/burma2d/update -H 'Content-Type: application/json' -d '{...}'"
echo ""
