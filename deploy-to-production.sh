#!/bin/bash

# 🚀 Production Deployment Script for go-server
# Deploy to: 76.13.218.203:22 (ai.a889.cloud)
# Last Updated: 2026-03-23

set -e

echo "🚀 AI Content Moderation System - Production Deployment"
echo "═══════════════════════════════════════════════════════════"

# Color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
REPO_URL="https://github.com/XavierMary56/go-server.git"
DEPLOY_DIR="/opt/moderation"
SERVICE_NAME="moderation"
DOMAIN="ai.a889.cloud"
SERVER_IP="76.13.218.203"

# Step 1: Check prerequisites
echo -e "\n${BLUE}[Step 1/6]${NC} Checking prerequisites..."

if ! command -v git &> /dev/null; then
    echo -e "${RED}❌ Git not found${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Git found${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}⚠️  Docker not found (will use binary deployment)${NC}"
    DEPLOYMENT_METHOD="binary"

    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go not found and no Docker - cannot proceed${NC}"
        exit 1
    fi
else
    DEPLOYMENT_METHOD="docker"
    echo -e "${GREEN}✅ Docker found - will use Docker deployment${NC}"
fi

# Step 2: Create deployment directory
echo -e "\n${BLUE}[Step 2/6]${NC} Setting up deployment directory..."

if [ ! -d "$DEPLOY_DIR" ]; then
    echo "Creating $DEPLOY_DIR..."
    sudo mkdir -p "$DEPLOY_DIR"
    sudo chown "$USER:$USER" "$DEPLOY_DIR"
fi

cd "$DEPLOY_DIR"
echo -e "${GREEN}✅ Deployment directory ready: $DEPLOY_DIR${NC}"

# Step 3: Clone or update repository
echo -e "\n${BLUE}[Step 3/6]${NC} Fetching latest code from GitHub..."

if [ -d ".git" ]; then
    echo "Repository exists, pulling latest changes..."
    git fetch origin
    git reset --hard origin/main
else
    echo "Cloning repository..."
    git clone "$REPO_URL" .
fi

echo -e "${GREEN}✅ Code updated to latest version${NC}"

# Step 4: Configure environment
echo -e "\n${BLUE}[Step 4/6]${NC} Preparing configuration..."

if [ ! -f ".env" ]; then
    if [ -f ".env.production" ]; then
        cp .env.production .env
        echo -e "${GREEN}✅ .env created from .env.production${NC}"
    else
        echo -e "${RED}❌ Neither .env nor .env.production found${NC}"
        echo "Please create .env file with required configuration:"
        echo "  ANTHROPIC_API_KEY=your_key"
        echo "  ENABLED_AUTH=true"
        echo "  ALLOWED_KEYS=your_project_keys"
        exit 1
    fi
fi

# Check if ANTHROPIC_API_KEY is configured
if ! grep -q "ANTHROPIC_API_KEY=" .env || grep "^ANTHROPIC_API_KEY=$" .env > /dev/null; then
    echo -e "${RED}❌ ANTHROPIC_API_KEY not configured in .env${NC}"
    echo "Please edit .env and set ANTHROPIC_API_KEY"
    exit 1
fi

echo -e "${GREEN}✅ Configuration verified${NC}"

# Step 5: Deploy based on method
echo -e "\n${BLUE}[Step 5/6]${NC} Deploying service (using $DEPLOYMENT_METHOD)..."

if [ "$DEPLOYMENT_METHOD" = "docker" ]; then
    echo "Building and starting Docker containers..."
    docker-compose up -d

    echo "Waiting for service to start..."
    sleep 5

    SERVICE_RUNNING=$(docker-compose ps | grep "Up" | wc -l)
    if [ "$SERVICE_RUNNING" -gt 0 ]; then
        echo -e "${GREEN}✅ Docker containers started successfully${NC}"
    else
        echo -e "${RED}❌ Docker containers failed to start${NC}"
        docker-compose logs
        exit 1
    fi
else
    echo "Building binary..."
    go build -o moderation-server ./cmd/server

    echo "Setting up systemd service..."
    sudo cp deploy/moderation.service /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable moderation
    sudo systemctl start moderation

    sleep 2

    if sudo systemctl is-active --quiet moderation; then
        echo -e "${GREEN}✅ Service started successfully${NC}"
    else
        echo -e "${RED}❌ Service failed to start${NC}"
        sudo journalctl -u moderation -n 20
        exit 1
    fi
fi

# Step 6: Verify deployment
echo -e "\n${BLUE}[Step 6/6]${NC} Verifying deployment..."

# Try health check
HEALTH_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/v1/health 2>/dev/null || echo "000")

if [ "$HEALTH_RESPONSE" = "200" ]; then
    echo -e "${GREEN}✅ Health check passed${NC}"
else
    echo -e "${YELLOW}⚠️  Health check returned $HEALTH_RESPONSE (service may still be starting)${NC}"
fi

# Summary
echo -e "\n${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ DEPLOYMENT COMPLETED SUCCESSFULLY!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"

echo -e "\n📊 Deployment Information:"
echo -e "  ${BLUE}Server IP${NC}:     $SERVER_IP:22"
echo -e "  ${BLUE}Domain${NC}:        $DOMAIN"
echo -e "  ${BLUE}Deploy Dir${NC}:    $DEPLOY_DIR"
echo -e "  ${BLUE}Method${NC}:        $DEPLOYMENT_METHOD"
echo -e "  ${BLUE}Service${NC}:       $SERVICE_NAME"

echo -e "\n🔍 Next Steps:"
echo -e "  1. ${BLUE}Verify service:${NC}   curl http://localhost:8080/v1/health"
echo -e "  2. ${BLUE}Check logs:${NC}       bash monitor.sh logs -n 50"
echo -e "  3. ${BLUE}Test API:${NC}        bash monitor.sh status"
echo -e "  4. ${BLUE}Configure domain${NC}: Update DNS records for $DOMAIN → $SERVER_IP"
echo -e "  5. ${BLUE}Setup HTTPS:${NC}     Configure Nginx and SSL certificate"

echo -e "\n📖 Documentation:"
echo -e "  ${BLUE}Quick Start${NC}:    docs/02-deployment/API_AND_DEPLOYMENT.md"
echo -e "  ${BLUE}Scripts Guide${NC}:  docs/04-operations/SCRIPTS_GUIDE.md"
echo -e "  ${BLUE}Monitoring${NC}:     docs/04-operations/AUTH_AND_MONITORING.md"

echo -e "\n💡 Useful Commands:"
echo -e "  ${BLUE}View logs${NC}:        bash monitor.sh logs -n 100"
echo -e "  ${BLUE}Monitor metrics${NC}:   bash monitor.sh metrics"
echo -e "  ${BLUE}Manage keys${NC}:      bash manage-keys.sh list"
echo -e "  ${BLUE}Service status${NC}:   bash monitor.sh status"

echo -e "\n${GREEN}Happy serving! 🎉${NC}"
