#!/bin/bash
# High-Performance Server Startup Script for 10,000+ Concurrent Users
# Burma2D Lottery Server Optimization

set -e

echo "🚀 Starting Burma2D Server with High-Concurrency Optimizations..."

# Step 1: Check and increase file descriptor limits
echo "📋 Checking system limits..."

CURRENT_ULIMIT=$(ulimit -n)
REQUIRED_ULIMIT=30000

if [ "$CURRENT_ULIMIT" -lt "$REQUIRED_ULIMIT" ]; then
    echo "⚠️  Current file descriptor limit: $CURRENT_ULIMIT (too low!)"
    echo "🔧 Attempting to increase to $REQUIRED_ULIMIT..."
    ulimit -n $REQUIRED_ULIMIT 2>/dev/null || {
        echo "❌ Failed to increase ulimit. Run as root or edit /etc/security/limits.conf"
        echo ""
        echo "Add these lines to /etc/security/limits.conf:"
        echo "* soft nofile 30000"
        echo "* hard nofile 50000"
        echo ""
        echo "Then logout and login again, or run:"
        echo "sudo systemctl restart burma2d.service"
        exit 1
    }
    echo "✅ File descriptor limit increased to $(ulimit -n)"
else
    echo "✅ File descriptor limit: $CURRENT_ULIMIT (sufficient)"
fi

# Step 2: Set Go runtime optimizations
echo "⚡ Setting Go runtime optimizations..."

# Use all available CPU cores
export GOMAXPROCS=$(nproc)
echo "✅ GOMAXPROCS=$GOMAXPROCS (using all CPU cores)"

# Aggressive garbage collection for lower memory usage
export GOGC=50
echo "✅ GOGC=50 (aggressive GC, lower memory)"

# Disable Go's own CPU profiling overhead
export GODEBUG=gctrace=0
echo "✅ GODEBUG=gctrace=0 (no GC trace overhead)"

# Step 3: Check TCP/IP kernel settings
echo "🌐 Checking network settings..."

check_sysctl() {
    local key=$1
    local recommended=$2
    local current=$(sysctl -n $key 2>/dev/null || echo "0")
    
    if [ "$current" -lt "$recommended" ]; then
        echo "⚠️  $key = $current (recommended: $recommended)"
        echo "   Run: sudo sysctl -w $key=$recommended"
    else
        echo "✅ $key = $current"
    fi
}

check_sysctl "net.core.somaxconn" 4096
check_sysctl "net.ipv4.tcp_max_syn_backlog" 8192
check_sysctl "net.ipv4.ip_local_port_range" 1024

# Step 4: Check if server binary exists
if [ ! -f "./burma2d-server" ]; then
    echo "❌ Server binary not found! Building..."
    go build -o burma2d-server main.go
    echo "✅ Server built successfully"
fi

# Step 5: Set database path
export DATABASE_PATH="${DATABASE_PATH:-./burma2d.db}"
echo "✅ Database: $DATABASE_PATH"

# Step 6: Start server with optimizations
echo ""
echo "🎯 Starting server optimized for 10,000+ concurrent users..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Run server (it will use GOMAXPROCS and other env vars set above)
./burma2d-server

# Note: If you want to run in background, use:
# nohup ./burma2d-server > server.log 2>&1 &
# echo $! > server.pid
# echo "✅ Server started in background (PID: $(cat server.pid))"
