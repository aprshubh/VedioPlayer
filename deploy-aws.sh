#!/usr/bin/env bash
# ==============================================================================
# DYUET — Automated 1-Click AWS Deployment Script
# Tested on: Ubuntu 22.04 / 24.04 LTS (AWS EC2 / Lightsail)
# ==============================================================================

set -e

echo "=========================================="
echo "🎬 Deploying DYUET on AWS Server..."
echo "=========================================="

# 1. Install Docker & Docker Compose if not already installed
if ! command -v docker &> /dev/null; then
    echo "📦 Installing Docker..."
    sudo apt-get update -y
    sudo apt-get install -y ca-certificates curl gnupg lsb-release
    sudo mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt-get update -y
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    sudo systemctl enable docker
    sudo systemctl start docker
    sudo usermod -aG docker $USER
    echo "✅ Docker installed successfully."
fi

# 2. Build & Launch Containers
echo "🚀 Building and launching DYUET stack (Postgres + Redis + Backend + Frontend)..."
sudo docker compose down --remove-orphans || true
sudo docker compose build --no-cache
sudo docker compose up -d

echo ""
echo "=========================================="
echo "✨ DYUET is now LIVE on your AWS Server!"
echo "=========================================="
PUBLIC_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || echo "your-server-ip")
echo "🌐 Access your app at: http://${PUBLIC_IP}"
echo "📡 Health check:       http://${PUBLIC_IP}/api/health"
echo "=========================================="
