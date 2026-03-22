#!/bin/bash
# ══════════════════════════════════════════════════════════════
#  push-go-to-github.sh — 推送到 go-server 仓库
#  用法：bash push-go-to-github.sh
# ══════════════════════════════════════════════════════════════
set -e

GITHUB_TOKEN="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"
GITHUB_USER="XavierMary56"
REPO_NAME="go-server"
REPO_URL="https://${GITHUB_TOKEN}@github.com/${GITHUB_USER}/${REPO_NAME}.git"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🚀 推送 Go 审核服务 → ${GITHUB_USER}/${REPO_NAME}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

cd "${SCRIPT_DIR}"

echo "[1/3] 初始化 git..."
git init -q
git config user.name  "${GITHUB_USER}"
git config user.email "${GITHUB_USER}@users.noreply.github.com"

echo "[2/3] 提交代码..."
git add -A
git rm --cached push-go-to-github.sh 2>/dev/null || true
git commit -q -m "🎉 初始提交：AI 内容审核服务 Go 版 v2.0

- 多模型自动轮换（Sonnet 4 / Haiku 4.5 / Opus 4）
- 同步审核 + 异步 Webhook 双模式
- Docker Compose 一键部署（含 Redis）
- PHP YAF 客户端兼容原接口
- 完整 README 部署文档"

echo "[3/3] 推送到 GitHub..."
if git remote | grep -q "^origin$"; then
  git remote set-url origin "${REPO_URL}"
else
  git remote add origin "${REPO_URL}"
fi
git branch -M main
git push -u origin main --force -q

# 清除 Token
git remote set-url origin "https://github.com/${GITHUB_USER}/${REPO_NAME}.git"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ 推送成功！"
echo "  🔗 https://github.com/${GITHUB_USER}/${REPO_NAME}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
