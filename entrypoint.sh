#!/bin/bash
# Entrypoint script to start both frontend and backend services
# Usage: ./entrypoint.sh [options]
# Options:
#   --no-frontend   Start only backend
#   --no-backend     Start only frontend

set -e

# Configuration
BACKEND_PORT="${BACKEND_PORT:-8088}"
FRONTEND_PORT="${FRONTEND_PORT:-3088}"
BACKEND_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$BACKEND_DIR/ui"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default options
START_BACKEND=true
START_FRONTEND=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-frontend)
            START_FRONTEND=false
            shift
            ;;
        --no-backend)
            START_BACKEND=false
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --no-frontend    Start only backend"
            echo "  --no-backend     Start only frontend"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  BACKEND_PORT      Backend port (default: 8088)"
            echo "  FRONTEND_PORT     Frontend port (default: 3088)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Function to check if a port is in use
check_port() {
    local port=$1
    if command -v lsof &> /dev/null; then
        lsof -i :$port &> /dev/null
    elif command -v netstat &> /dev/null; then
        netstat -tuln | grep -q ":$port "
    else
        # Fallback: assume port is available
        return 1
    fi
}

# Function to start backend
start_backend() {
    echo -e "${GREEN}[Backend]${NC} Starting Go server..."
    
    cd "$BACKEND_DIR"
    
    # Check if config exists
    if [ ! -f "configs/config.yaml" ]; then
        echo -e "${YELLOW}[Backend]${NC} Warning: configs/config.yaml not found, using defaults"
    fi
    
    # Start backend in background
    go run cmd/server/main.go &
    BACKEND_PID=$!
    
    echo -e "${GREEN}[Backend]${NC} Started (PID: $BACKEND_PID) on port $BACKEND_PORT"
}

# Function to start frontend
start_frontend() {
    echo -e "${GREEN}[Frontend]${NC} Starting React development server..."
    
    cd "$FRONTEND_DIR"
    
    # Check if node_modules exists
    if [ ! -d "node_modules" ]; then
        echo -e "${YELLOW}[Frontend]${NC} Installing dependencies..."
        npm install
    fi
    
    # Export environment variables for Vite
    export FRONTEND_PORT
    export BACKEND_URL="http://localhost:$BACKEND_PORT"
    
    # Start frontend in background
    FRONTEND_PORT=$FRONTEND_PORT BACKEND_URL=http://localhost:$BACKEND_PORT npm run dev &
    FRONTEND_PID=$!
    
    echo -e "${GREEN}[Frontend]${NC} Started (PID: $FRONTEND_PID) on port $FRONTEND_PORT"
}

# Function to handle shutdown
cleanup() {
    echo ""
    echo -e "${YELLOW}[Entrypoint]${NC} Shutting down services..."
    
    if [ -n "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
        echo -e "${GREEN}[Backend]${NC} Stopped"
    fi
    
    if [ -n "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
        echo -e "${GREEN}[Frontend]${NC} Stopped"
    fi
    
    exit 0
}

# Trap signals for graceful shutdown
trap cleanup SIGINT SIGTERM

# Start services
echo -e "${GREEN}[Entrypoint]${NC} Starting trace-point services..."
echo ""

if [ "$START_BACKEND" = true ]; then
    start_backend
else
    echo -e "${YELLOW}[Backend]${NC} Skipped (--no-backend flag)"
fi

if [ "$START_FRONTEND" = true ]; then
    start_frontend
else
    echo -e "${YELLOW}[Frontend]${NC} Skipped (--no-frontend flag)"
fi

echo ""
echo -e "${GREEN}[Entrypoint]${NC} All services started successfully!"
echo ""
echo "Access points:"
echo "  - Frontend: http://localhost:$FRONTEND_PORT"
echo "  - Backend:  http://localhost:$BACKEND_PORT"
echo "  - Health:   http://localhost:$BACKEND_PORT/health"
echo ""
echo "Press Ctrl+C to stop all services"

# Wait for any process to exit
wait
