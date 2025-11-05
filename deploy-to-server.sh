#!/bin/bash
# Deploy Burma2D Server to Production
# This script builds the Linux binary and provides instructions for deployment

set -e

echo "🏗️  Building Linux binary..."
cd "$(dirname "$0")"

# Build for Linux (server is Linux)
GOOS=linux GOARCH=amd64 go build -o burma2d-server-linux main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "📦 Binary created: burma2d-server-linux"
    echo ""
    echo "📝 Deployment Steps:"
    echo "===================="
    echo ""
    echo "1. Upload binary to server:"
    echo "   scp burma2d-server-linux root@vmi2760068.contaboserver.net:/www/wwwroot/api.shwemyanmar2d.us/burma2dserver/"
    echo ""
    echo "2. SSH to server:"
    echo "   ssh root@vmi2760068.contaboserver.net"
    echo ""
    echo "3. Navigate to directory:"
    echo "   cd /www/wwwroot/api.shwemyanmar2d.us/burma2dserver/"
    echo ""
    echo "4. Make executable:"
    echo "   chmod +x burma2d-server-linux"
    echo ""
    echo "5. Stop current service:"
    echo "   systemctl stop masterserver"
    echo ""
    echo "6. Backup old binary:"
    echo "   mv masterserver masterserver.backup-\$(date +%Y%m%d-%H%M%S)"
    echo ""
    echo "7. Replace with new binary:"
    echo "   mv burma2d-server-linux masterserver"
    echo ""
    echo "8. Start service:"
    echo "   systemctl start masterserver"
    echo ""
    echo "9. Check status:"
    echo "   systemctl status masterserver"
    echo ""
    echo "10. View logs:"
    echo "   journalctl -u masterserver -f"
    echo ""
    echo "11. Test API:"
    echo "   curl https://api.shwemyanmar2d.us/api/burma2d/history | jq '. | length'"
    echo "   curl https://api.shwemyanmar2d.us/api/burma2d/threed | jq '. | length'"
    echo ""
else
    echo "❌ Build failed!"
    exit 1
fi
