#!/bin/bash
# ══════════════════════════════════════════════════════════
# 完整部署脚本
# 使用方式：bash deploy.sh
# ══════════════════════════════════════════════════════════

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}  AI 内容审核服务 - 生产部署${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# 1. 环境检查
echo -e "${YELLOW}[1/6]${NC} 环境检查..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker 未安装${NC}"
    exit 1
fi
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose 未安装${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker 和 Docker Compose 已安装${NC}"

# 2. 配置文件准备
echo -e "${YELLOW}[2/6]${NC} 配置文件准备..."
if [ ! -f ".env" ]; then
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo -e "${GREEN}✓ 已复制 .env.example 为 .env${NC}"
    else
        echo -e "${RED}❌ 找不到 .env.example 或 .env${NC}"
        exit 1
    fi
fi

# 检查必填配置
if ! grep -q "ANTHROPIC_API_KEY=" .env; then
    echo -e "${RED}❌ .env 中缺少 ANTHROPIC_API_KEY${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 配置文件检查通过${NC}"

# 3. 编译应用（如果需要）
echo -e "${YELLOW}[3/6]${NC} 应用构建..."
if [ -f "Dockerfile" ]; then
    echo -e "${GREEN}✓ 使用 Docker 构建（在 Docker Compose 中自动进行）${NC}"
else
    echo -e "${RED}❌ 找不到 Dockerfile${NC}"
    exit 1
fi

# 4. 目录权限设置
echo -e "${YELLOW}[4/6]${NC} 目录权限设置..."
mkdir -p logs/audit logs/metrics
chmod 755 logs logs/audit logs/metrics
echo -e "${GREEN}✓ 日志目录已创建${NC}"

# 5. 启动服务
echo -e "${YELLOW}[5/6]${NC} 启动服务..."
docker-compose down 2>/dev/null || true
docker-compose up -d

echo -e "${GREEN}✓ 容器已启动${NC}"
sleep 2

# 6. 服务验证
echo -e "${YELLOW}[6/6]${NC} 服务健康检查..."
if curl -s http://localhost:888/v1/health | grep -q "ok"; then
    echo -e "${GREEN}✓ 服务已启动成功${NC}"
else
    echo -e "${RED}❌ 服务启动失败，请检查日志${NC}"
    docker-compose logs moderation
    exit 1
fi

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ 部署完成！${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

echo -e "${YELLOW}📋 服务信息：${NC}"
echo "  - API 地址（本地）：http://localhost:888"
echo "  - API 地址（生产）：https://zyaokkmo.cc"
echo "  - 健康检查：curl -s http://localhost:888/v1/health"
echo ""

echo -e "${YELLOW}📝 日志管理：${NC}"
echo "  - 应用日志：docker-compose logs -f moderation"
echo "  - 审计日志：tail -f logs/audit_*.log"
echo ""

echo -e "${YELLOW}🔍 测试 API：${NC}"
echo "  - 基础测试："
echo "    curl http://localhost:888/v1/health"
echo ""
echo "  - 带密钥测试（如果启用了鉴权）："
echo "    curl -H 'X-Project-Key: sk-proj-forum-a-k3j9x2m1' \\"
echo "         http://localhost:888/v1/models"
echo ""
echo "  - 审核请求："
echo "    curl -X POST http://localhost:888/v1/moderate \\"
echo "         -H 'Content-Type: application/json' \\"
echo "         -H 'X-Project-Key: sk-proj-forum-a-k3j9x2m1' \\"
echo "         -d '{\"content\":\"测试内容\",\"type\":\"comment\"}'"
echo ""

echo -e "${YELLOW}🚀 下一步：${NC}"
echo "  1. 配置 Nginx 反向代理（见 deploy/nginx.conf）"
echo "  2. 绑定域名 zyaokkmo.cc"
echo "  3. 配置 HTTPS 证书（Let's Encrypt）"
echo "  4. 设置监控和告警"
