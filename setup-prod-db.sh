#!/bin/bash
set -e

echo "=============================================="
echo "🚀 ThaiMaster2D Production Database Setup"
echo "=============================================="
echo ""

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   echo "⚠️  Please run this script as a normal user (not root)"
   echo "   The script will ask for sudo password when needed"
   exit 1
fi

echo "📦 Step 1: Installing PostgreSQL..."
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib
echo "✅ PostgreSQL installed"
echo ""

echo "🔌 Step 2: Starting PostgreSQL service..."
sudo systemctl start postgresql
sudo systemctl enable postgresql
echo "✅ PostgreSQL service started and enabled"
echo ""

echo "🗄️  Step 3: Creating database 'thaimaster2d'..."
sudo -u postgres createdb thaimaster2d 2>/dev/null || echo "   (Database already exists, skipping)"
echo "✅ Database ready"
echo ""

echo "🔐 Step 4: Setting postgres user password..."
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" > /dev/null 2>&1
echo "✅ Password set to 'postgres'"
echo ""

echo "⚙️  Step 5: Configuring PostgreSQL authentication..."
# Backup original pg_hba.conf
sudo cp /etc/postgresql/*/main/pg_hba.conf /etc/postgresql/*/main/pg_hba.conf.backup 2>/dev/null || true
# Change peer to md5 for postgres user
sudo sed -i 's/local.*all.*postgres.*peer/local   all             postgres                                md5/' /etc/postgresql/*/main/pg_hba.conf
sudo systemctl restart postgresql
echo "✅ Authentication configured (md5)"
echo ""

echo "🌍 Step 6: Setting DATABASE_URL environment variable..."
if grep -q "DATABASE_URL" ~/.bashrc; then
    echo "   DATABASE_URL already exists in ~/.bashrc"
else
    echo "export DATABASE_URL='postgres://postgres:postgres@localhost:5432/thaimaster2d?sslmode=disable'" >> ~/.bashrc
    echo "✅ DATABASE_URL added to ~/.bashrc"
fi
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/thaimaster2d?sslmode=disable'
echo ""

echo "🧪 Step 7: Testing database connection..."
if PGPASSWORD=postgres psql -h localhost -U postgres -d thaimaster2d -c "SELECT version();" > /dev/null 2>&1; then
    echo "✅ Database connection successful!"
    PGPASSWORD=postgres psql -h localhost -U postgres -d thaimaster2d -c "SELECT version();" | head -3
else
    echo "❌ Database connection failed"
    echo "   Please check the error messages above"
    exit 1
fi
echo ""

echo "=============================================="
echo "✨ Database Setup Complete!"
echo "=============================================="
echo ""
echo "📋 Next steps:"
echo ""
echo "1. Reload your shell environment:"
echo "   source ~/.bashrc"
echo ""
echo "2. Go to your server directory and rebuild:"
echo "   cd /path/to/your/Go"
echo "   go build -o thaimaster2d-server"
echo ""
echo "3. Stop old server if running:"
echo "   pkill thaimaster2d-server"
echo ""
echo "4. Start the server:"
echo "   ./thaimaster2d-server"
echo ""
echo "   OR run in background:"
echo "   nohup ./thaimaster2d-server > server.log 2>&1 &"
echo ""
echo "5. Check the logs for:"
echo "   ✅ Database connected successfully!"
echo "   ✅ All database modules initialized!"
echo ""
echo "6. Test the endpoints:"
echo "   curl http://213.136.80.25:4545/admin"
echo "   curl http://213.136.80.25:4545/api/paper/types"
echo ""
echo "=============================================="
