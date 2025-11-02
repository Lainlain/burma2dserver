# Test Firebase OAuth authentication endpoint
# This demonstrates how to send a Google ID token to the server for verification

Write-Host "🔐 Testing Firebase OAuth Authentication..." -ForegroundColor Cyan
Write-Host ""

# Test 1: Development mode (no token verification)
Write-Host "Test 1: Development mode authentication..." -ForegroundColor Yellow
$body1 = @{
    id_token = "test-token"
    email = "test@example.com"
    username = "Test User"
    photo_url = "https://example.com/photo.jpg"
} | ConvertTo-Json

try {
    $response1 = Invoke-RestMethod -Uri "http://localhost:4545/api/chat/auth/google" `
        -Method POST `
        -ContentType "application/json" `
        -Body $body1
    Write-Host "Response: $($response1 | ConvertTo-Json)" -ForegroundColor Green
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Test 2: Invalid token (will fail if GOOGLE_OAUTH_CLIENT_ID is set)
Write-Host "Test 2: Invalid token authentication (should fail in production)..." -ForegroundColor Yellow
$body2 = @{
    id_token = "invalid-token"
    email = "hacker@example.com"
    username = "Hacker"
    photo_url = "https://example.com/hacker.jpg"
} | ConvertTo-Json

try {
    $response2 = Invoke-RestMethod -Uri "http://localhost:4545/api/chat/auth/google" `
        -Method POST `
        -ContentType "application/json" `
        -Body $body2
    Write-Host "Response: $($response2 | ConvertTo-Json)" -ForegroundColor Green
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

Write-Host "✅ Tests completed!" -ForegroundColor Green
Write-Host ""
Write-Host "📝 Notes:" -ForegroundColor Cyan
Write-Host "  - In development mode (no GOOGLE_OAUTH_CLIENT_ID), all requests are accepted"
Write-Host "  - In production mode, only valid Google ID tokens will be accepted"
Write-Host "  - Real Android client should send actual Google Sign-In ID token"
Write-Host ""
Write-Host "🔧 To enable production mode:" -ForegroundColor Cyan
Write-Host '  $env:GOOGLE_OAUTH_CLIENT_ID="YOUR-CLIENT-ID.apps.googleusercontent.com"'
Write-Host "  .\burma2d-server.exe"
