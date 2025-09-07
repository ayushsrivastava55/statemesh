#!/bin/bash

# Local development setup script for Cosmos State Mesh
set -e

echo "🚀 Setting up Cosmos State Mesh for local development..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

echo "✅ Go found: $(go version)"

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Please run this script from the project root directory"
    exit 1
fi

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod tidy
go mod download

# Build the application
echo "🔨 Building State Mesh..."
make build

echo "✅ Build completed!"

# Check for local services
echo "🔍 Checking for required services..."

# Check PostgreSQL
if command -v psql &> /dev/null; then
    echo "✅ PostgreSQL client found"
else
    echo "⚠️  PostgreSQL client not found. Install with: brew install postgresql"
fi

# Check if PostgreSQL is running
if pg_isready -h localhost -p 5432 &> /dev/null; then
    echo "✅ PostgreSQL is running"
else
    echo "⚠️  PostgreSQL is not running. Start with: brew services start postgresql"
fi

# Create database if it doesn't exist
echo "🗄️  Setting up database..."
createdb statemesh 2>/dev/null || echo "Database 'statemesh' already exists or couldn't be created"

# Run migrations
if [ -f "bin/state-mesh" ]; then
    echo "🔄 Running database migrations..."
    # Note: You'll need to implement a migrate command or run migrations manually
    echo "⚠️  Please run PostgreSQL migrations manually from migrations/postgres/001_initial_schema.sql"
fi

echo ""
echo "🎉 Setup complete!"
echo ""
echo "To run the application:"
echo "  1. Start PostgreSQL: brew services start postgresql"
echo "  2. Start the API server: ./bin/state-mesh serve"
echo "  3. Start the ingester: ./bin/state-mesh ingest"
echo ""
echo "API endpoints will be available at:"
echo "  - GraphQL: http://localhost:8080/graphql"
echo "  - GraphQL Playground: http://localhost:8080/playground"
echo "  - REST API: http://localhost:8081/api/v1"
echo "  - Metrics: http://localhost:8082/metrics"
