#!/bin/bash

# Test Firebase OAuth authentication endpoint
# This demonstrates how to send a Google ID token to the server for verification

echo "🔐 Testing Firebase OAuth Authentication..."
echo ""

# Test 1: Development mode (no token verification)
echo "Test 1: Development mode authentication..."
curl -X POST http://localhost:4545/api/chat/auth/google \
  -H "Content-Type: application/json" \
  -d '{
    "id_token": "test-token",
    "email": "test@example.com",
    "username": "Test User",
    "photo_url": "https://example.com/photo.jpg"
  }'
echo -e "\n"

# Test 2: Invalid token (will fail if GOOGLE_OAUTH_CLIENT_ID is set)
echo "Test 2: Invalid token authentication (should fail in production)..."
curl -X POST http://localhost:4545/api/chat/auth/google \
  -H "Content-Type: application/json" \
  -d '{
    "id_token": "invalid-token",
    "email": "hacker@example.com",
    "username": "Hacker",
    "photo_url": "https://example.com/hacker.jpg"
  }'
echo -e "\n"

echo ""
echo "✅ Tests completed!"
echo ""
echo "📝 Notes:"
echo "  - In development mode (no GOOGLE_OAUTH_CLIENT_ID), all requests are accepted"
echo "  - In production mode, only valid Google ID tokens will be accepted"
echo "  - Real Android client should send actual Google Sign-In ID token"
echo ""
echo "🔧 To enable production mode:"
echo "  export GOOGLE_OAUTH_CLIENT_ID=\"YOUR-CLIENT-ID.apps.googleusercontent.com\""
echo "  ./burma2d-server.exe"
