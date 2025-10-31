#!/bin/bash

echo "==================================="
echo "🔍 Database Connection Diagnostics"
echo "==================================="
echo ""

# Check if PostgreSQL is installed
echo "1️⃣ Checking if PostgreSQL is installed..."
if command -v psql &> /dev/null; then
    echo "✅ PostgreSQL client is installed"
    psql --version
else
    echo "❌ PostgreSQL client is NOT installed"
    echo "   Install it with: sudo apt-get install postgresql-client"
fi
echo ""

# Check if PostgreSQL service is running
echo "2️⃣ Checking if PostgreSQL service is running..."
if systemctl is-active --quiet postgresql; then
    echo "✅ PostgreSQL service is running"
    systemctl status postgresql --no-pager | head -5
else
    echo "❌ PostgreSQL service is NOT running"
    echo "   Start it with: sudo systemctl start postgresql"
fi
echo ""

# Check DATABASE_URL environment variable
echo "3️⃣ Checking DATABASE_URL environment variable..."
if [ -z "$DATABASE_URL" ]; then
    echo "❌ DATABASE_URL is NOT set"
    echo "   Set it with: export DATABASE_URL='postgres://postgres:postgres@localhost:5432/thaimaster2d?sslmode=disable'"
else
    echo "✅ DATABASE_URL is set to: $DATABASE_URL"
fi
echo ""

# Test database connection
echo "4️⃣ Testing database connection..."
if [ -z "$DATABASE_URL" ]; then
    # Try default connection
    echo "Testing default connection: postgres://postgres:postgres@localhost:5432/thaimaster2d"
    if PGPASSWORD=postgres psql -h localhost -U postgres -d thaimaster2d -c "SELECT version();" &> /dev/null; then
        echo "✅ Database connection successful!"
    else
        echo "❌ Cannot connect to database"
        echo "   Check if database 'thaimaster2d' exists"
    fi
else
    echo "Testing connection with DATABASE_URL..."
    if psql "$DATABASE_URL" -c "SELECT version();" &> /dev/null; then
        echo "✅ Database connection successful!"
    else
        echo "❌ Cannot connect to database"
    fi
fi
echo ""

# Check if database exists
echo "5️⃣ Checking if 'thaimaster2d' database exists..."
if PGPASSWORD=postgres psql -h localhost -U postgres -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw thaimaster2d; then
    echo "✅ Database 'thaimaster2d' exists"
else
    echo "❌ Database 'thaimaster2d' does NOT exist"
    echo "   Create it with: sudo -u postgres createdb thaimaster2d"
fi
echo ""

echo "==================================="
echo "📋 RECOMMENDATIONS:"
echo "==================================="
echo ""
echo "If database is not set up, run these commands on your production server:"
echo ""
echo "1. Install PostgreSQL:"
echo "   sudo apt-get update"
echo "   sudo apt-get install postgresql postgresql-contrib"
echo ""
echo "2. Start PostgreSQL:"
echo "   sudo systemctl start postgresql"
echo "   sudo systemctl enable postgresql"
echo ""
echo "3. Create database:"
echo "   sudo -u postgres createdb thaimaster2d"
echo ""
echo "4. Set database password (optional):"
echo "   sudo -u postgres psql -c \"ALTER USER postgres PASSWORD 'postgres';\""
echo ""
echo "5. Set environment variable (add to ~/.bashrc or /etc/environment):"
echo "   export DATABASE_URL='postgres://postgres:postgres@localhost:5432/thaimaster2d?sslmode=disable'"
echo ""
echo "6. Rebuild and restart the server:"
echo "   go build -o thaimaster2d-server"
echo "   ./thaimaster2d-server"
echo ""
