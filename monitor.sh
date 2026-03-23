#!/bin/bash
# ══════════════════════════════════════════════════════════
# 监控和日志管理脚本
# 使用方式：bash monitor.sh [命令]
# ══════════════════════════════════════════════════════════

set -e

# 配置
LOG_DIR="./logs"
AUDIT_LOG_DIR="./logs/audit"
RETENTION_DAYS=30

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 帮助文本
show_help() {
    cat << EOF
${BLUE}AI 内容审核服务 - 监控和日志管理${NC}

${YELLOW}用法：${NC} bash monitor.sh [命令] [选项]

${YELLOW}命令：${NC}
  status              查看服务状态和实时指标
  logs                查看实时应用日志
  audit               查看审计日志
  metrics             获取详细的性能指标
  health              执行健康检查
  clean               清理旧日志（保留 $RETENTION_DAYS 天）
  backup              备份审计日志
  alert               配置告警阈值
  help                显示此帮助信息

${YELLOW}示例：${NC}
  bash monitor.sh status
  bash monitor.sh logs -n 50
  bash monitor.sh audit --project forum_service
  bash monitor.sh metrics --export prometheus
  bash monitor.sh clean
EOF
}

# 检查服务状态
check_status() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}服务状态${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # 检查 Docker 容器
    if docker ps | grep -q moderation; then
        echo -e "${GREEN}✓ 容器状态：运行中${NC}"
        docker stats --no-stream moderation 2>/dev/null || echo "  无法获取资源使用情况"
    else
        echo -e "${RED}✗ 容器状态：停止${NC}"
        return 1
    fi

    # 健康检查
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}健康检查${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if response=$(curl -s http://localhost:8080/v1/health); then
        echo -e "${GREEN}✓ API 健康${NC}"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        echo -e "${RED}✗ API 无响应${NC}"
        return 1
    fi

    # 性能指标
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}性能指标${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if response=$(curl -s http://localhost:8080/v1/stats); then
        echo "$response" | jq '.data' 2>/dev/null || echo "$response"
    else
        echo -e "${RED}无法获取统计数据${NC}"
    fi
}

# 实时日志
tail_logs() {
    local lines=${1:-100}
    echo -e "${BLUE}最近 $lines 条应用日志：${NC}"
    docker-compose logs -f --tail=$lines moderation
}

# 审计日志查询
audit_logs() {
    local project=${1:-""}
    local days=${2:-1}

    echo -e "${BLUE}审计日志查询${NC}"
    echo "项目：${project:-所有}"
    echo "时间范围：最近 $days 天"
    echo ""

    if [ ! -d "$AUDIT_LOG_DIR" ]; then
        echo -e "${YELLOW}审计日志目录不存在：$AUDIT_LOG_DIR${NC}"
        return 1
    fi

    find "$AUDIT_LOG_DIR" -name "audit_*.log" -mtime -$days -exec ls -lh {} \;

    echo ""
    echo -e "${YELLOW}审计日志摘要：${NC}"

    # 统计认证尝试
    echo ""
    echo "认证尝试："
    grep -h '"event_type":"auth_attempt"' "$AUDIT_LOG_DIR"/audit_*.log 2>/dev/null | \
        jq -r '.project_id' 2>/dev/null | sort | uniq -c || echo "  无数据"

    # 统计 API 调用
    echo ""
    echo "API 调用统计（按项目）："
    grep -h '"event_type":"api_call"' "$AUDIT_LOG_DIR"/audit_*.log 2>/dev/null | \
        jq -r '.project_id' 2>/dev/null | sort | uniq -c || echo "  无数据"

    # 速率限制触发
    echo ""
    echo "速率限制触发："
    grep -h '"event_type":"rate_limit_exceeded"' "$AUDIT_LOG_DIR"/audit_*.log 2>/dev/null | \
        wc -l || echo "  无触发"
}

# 性能指标导出
export_metrics() {
    local format=${1:-json}
    echo -e "${BLUE}性能指标导出（格式：$format）${NC}"

    response=$(curl -s http://localhost:8080/v1/stats)

    if [ "$format" = "prometheus" ]; then
        # 转换为 Prometheus 格式
        echo "# HELP moderation_total_requests 总请求数"
        echo "# TYPE moderation_total_requests counter"
        echo "$response" | jq '.data.total_requests' 2>/dev/null | sed 's/^/moderation_total_requests /'

        echo "# HELP moderation_success_rate 成功率"
        echo "# TYPE moderation_success_rate gauge"
        echo "$response" | jq '.data.success_rate_percent' 2>/dev/null | sed 's/^/moderation_success_rate /'
    else
        # JSON 格式
        echo "$response" | jq '.data'
    fi
}

# 清理旧日志
cleanup_logs() {
    echo -e "${YELLOW}清理 ${RETENTION_DAYS} 天前的日志...${NC}"

    if [ -d "$LOG_DIR" ]; then
        find "$LOG_DIR" -name "*.log" -mtime +$RETENTION_DAYS -delete -print | \
            while read file; do
                echo -e "${GREEN}✓ 已删除：$file${NC}"
            done
    fi

    if [ -d "$AUDIT_LOG_DIR" ]; then
        find "$AUDIT_LOG_DIR" -name "*.log" -mtime +$RETENTION_DAYS -delete -print | \
            while read file; do
                echo -e "${GREEN}✓ 已删除：$file${NC}"
            done
    fi

    du -sh "$LOG_DIR" "$AUDIT_LOG_DIR" 2>/dev/null

    echo -e "${GREEN}✓ 日志清理完成${NC}"
}

# 备份审计日志
backup_logs() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_dir="backups/audit_backup_$timestamp"

    mkdir -p "$backup_dir"

    if [ -d "$AUDIT_LOG_DIR" ]; then
        cp -r "$AUDIT_LOG_DIR"/* "$backup_dir/"
        tar -czf "${backup_dir}.tar.gz" "$backup_dir"
        rm -rf "$backup_dir"
        echo -e "${GREEN}✓ 审计日志已备份：${backup_dir}.tar.gz${NC}"
    else
        echo -e "${YELLOW}审计日志目录不存在${NC}"
    fi
}

# 告警配置
setup_alerts() {
    echo -e "${BLUE}告警配置${NC}"
    cat << 'EOF'

配置告警，需要监控以下指标：

1. 错误率过高 (> 5%)
   condition: failed_requests / total_requests > 0.05
   action: 发送告警邮件

2. 响应延迟过高 (> 5000ms)
   condition: avg_latency_ms > 5000
   action: 触发自动扩容

3. 认证失败频繁 (> 10次/分钟)
   condition: auth_fail_count > 10
   action: 记录可疑 IP，可考虑封禁

4. 速率限制频繁触发 (> 100次/小时)
   condition: rate_limit_exceeded > 100
   action: 通知相关项目负责人

5. 磁盘空间不足 (< 10%)
   condition: free_space_percent < 10
   action: 自动清理旧日志

将以下脚本配置到 crontab：

# 每 5 分钟检查一次
*/5 * * * * bash /opt/moderation/monitor.sh check-alerts

# 每天凌晨 2 点清理日志
0 2 * * * bash /opt/moderation/monitor.sh clean

# 每周日备份审计日志
0 3 * * 0 bash /opt/moderation/monitor.sh backup
EOF
}

# 主逻辑
case "${1:-help}" in
    status)
        check_status
        ;;
    logs)
        shift
        tail_logs "$@"
        ;;
    audit)
        shift
        audit_logs "$@"
        ;;
    metrics)
        shift
        if [ -n "$1" ]; then
            export_metrics "$1"
        else
            export_metrics
        fi
        ;;
    health)
        curl -s http://localhost:8080/v1/health | jq '.'
        ;;
    clean)
        cleanup_logs
        ;;
    backup)
        backup_logs
        ;;
    alert)
        setup_alerts
        ;;
    help|*)
        show_help
        ;;
esac
