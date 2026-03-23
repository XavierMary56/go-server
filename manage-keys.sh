#!/bin/bash
# ══════════════════════════════════════════════════════════
# API 密钥管理工具
# 使用方式：bash manage-keys.sh [命令]
# ══════════════════════════════════════════════════════════

set -e

API_URL="${API_URL:-https://ai.a889.cloud}"
ADMIN_KEY="${ADMIN_KEY:-}"

# 颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_help() {
    cat << EOF
${BLUE}API 密钥管理工具${NC}

${YELLOW}用法：${NC} bash manage-keys.sh [命令] [参数]

${YELLOW}命令：${NC}
  list                   列出所有密钥和项目
  test <密钥>            测试密钥是否有效
  rate-test <密钥>       测试速率限制
  create <项目ID> <速率> 创建新的密钥
  rotate <旧密钥> <新密钥> 轮换密钥
  disable <密钥>         禁用密钥
  export-json            导出所有密钥配置（JSON）
  help                   显示此帮助

${YELLOW}参数说明：${NC}
  <项目ID>    项目标识符，如：forum_service
  <速率>      每分钟请求限制，如：300
  <密钥>      API 密钥，如：sk-proj-xxxxx

${YELLOW}示例：${NC}
  bash manage-keys.sh list
  bash manage-keys.sh test sk-proj-forum-a-k3j9x2m1
  bash manage-keys.sh create forum_service 300
  bash manage-keys.sh rate-test sk-proj-forum-a-k3j9x2m1
  bash manage-keys.sh rotate sk-proj-old-xxx sk-proj-new-yyy
EOF
}

# 列出所有密钥
list_keys() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}API 密钥列表${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    printf "${YELLOW}%-35s${NC} ${YELLOW}%-20s${NC} ${YELLOW}%-10s${NC}\n" "密钥（隐藏）" "项目 ID" "速率限制"
    echo "──────────────────────────────────────────────────────────────────────"

    # 从 .env 文件读取密钥
    if [ -f ".env" ]; then
        grep "^ALLOWED_KEYS=" .env | cut -d'=' -f2- | tr ',' '\n' | while read line; do
            if [ -z "$line" ]; then continue; fi

            PROJECT=$(echo "$line" | cut -d'|' -f1)
            KEY=$(echo "$line" | cut -d'|' -f2)
            RATE=$(echo "$line" | cut -d'|' -f3)

            # 隐藏密钥中间部分
            MASKED_KEY=$(echo "$KEY" | sed 's/^\(..\{4\}\).*\(..\{4\}$\)/\1****\2/')

            printf "%-35s %-20s %-10s\n" "$MASKED_KEY" "$PROJECT" "$RATE"
        done
    else
        echo -e "${RED}❌ .env 文件不存在${NC}"
    fi

    echo ""
}

# 测试密钥有效性
test_key() {
    local key=$1
    if [ -z "$key" ]; then
        echo -e "${RED}❌ 请提供密钥${NC}"
        return 1
    fi

    echo -e "${YELLOW}测试密钥：$key${NC}"

    # 调用健康检查接口
    response=$(curl -s -w "\n%{http_code}" \
        -H "X-Project-Key: $key" \
        "${API_URL}/v1/health")

    http_code=$(tail -n1 <<< "$response")
    body=$(head -n-1 <<< "$response")

    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}✓ 密钥有效${NC}"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ 密钥无效或已过期${NC}"
        echo "HTTP 状态码：$http_code"
        echo "响应：$body"
        return 1
    fi
}

# 测试速率限制
rate_test() {
    local key=$1
    local requests=${2:-310}

    if [ -z "$key" ]; then
        echo -e "${RED}❌ 请提供密钥${NC}"
        return 1
    fi

    echo -e "${YELLOW}速率限制测试（发送 $requests 个请求）${NC}"
    echo "使用密钥：$key"
    echo ""

    local success=0
    local failed=0
    local rate_limited=0

    for i in $(seq 1 $requests); do
        response=$(curl -s -w "\n%{http_code}" \
            -X POST "${API_URL}/v1/moderate" \
            -H "Content-Type: application/json" \
            -H "X-Project-Key: $key" \
            -d '{"content":"测试内容","type":"comment"}')

        http_code=$(tail -n1 <<< "$response")

        case "$http_code" in
            200)
                ((success++))
                ;;
            429)
                ((rate_limited++))
                ;;
            *)
                ((failed++))
                ;;
        esac

        if [ $((i % 50)) -eq 0 ]; then
            echo "已发送 $i 个请求... (成功: $success, 限流: $rate_limited, 出错: $failed)"
        fi
    done

    echo ""
    echo -e "${BLUE}测试结果：${NC}"
    echo "总请求数：$requests"
    echo -e "${GREEN}成功：$success${NC}"
    echo -e "${YELLOW}限流：$rate_limited${NC}"
    echo -e "${RED}出错：$failed${NC}"

    if [ $rate_limited -gt 0 ]; then
        echo ""
        echo -e "${YELLOW}⚠️ 触发了速率限制！考虑增加配额：${NC}"
        echo "  ALLOWED_KEYS=...|$key|<更高的数字|"
    fi
}

# 生成新密钥
generate_key() {
    local project=$1
    local rate=$2

    if [ -z "$project" ] || [ -z "$rate" ]; then
        echo -e "${RED}❌ 用法：bash manage-keys.sh create <项目ID> <速率>${NC}"
        return 1
    fi

    # 生成随机密钥
    local key="sk-proj-$(echo -n "$project-$(date +%s%N)" | md5sum | cut -c1-24)"

    echo -e "${BLUE}生成新密钥${NC}"
    echo "项目 ID：$project"
    echo "速率限制：$rate 请求/分钟"
    echo ""
    echo -e "${GREEN}新密钥：${NC}$key"
    echo ""
    echo -e "${YELLOW}将以下内容添加到 .env 的 ALLOWED_KEYS：${NC}"
    echo "${project}|${key}|${rate}"
    echo ""
    echo -e "${YELLOW}或编辑 .env 文件，追加：${NC}"
    echo "ALLOWED_KEYS=....,${project}|${key}|${rate}"
    echo ""
    echo -e "${YELLOW}保存后重启服务：${NC}"
    echo "docker-compose restart moderation"
}

# 轮换密钥
rotate_key() {
    local old_key=$1
    local new_key=$2

    if [ -z "$old_key" ] || [ -z "$new_key" ]; then
        echo -e "${RED}❌ 用法：bash manage-keys.sh rotate <旧密钥> <新密钥>${NC}"
        return 1
    fi

    echo -e "${YELLOW}密钥轮换流程：${NC}"
    echo ""
    echo "1️⃣ 编辑 .env 文件，将旧密钥替换为新密钥"
    echo "2️⃣ 重启服务：docker-compose restart moderation"
    echo "3️⃣ 测试新密钥：bash manage-keys.sh test $new_key"
    echo "4️⃣ 通知客户端使用新密钥"
    echo "5️⃣ 等待完全切换后，从 .env 中删除旧密钥"
    echo ""
    echo -e "${GREEN}轮换步骤：${NC}"
    echo ""
    echo "$ nano .env"
    echo "# 修改 ALLOWED_KEYS，将 $old_key 改为 $new_key"
    echo ""
    echo "$ docker-compose restart moderation"
    echo "$ bash manage-keys.sh test $new_key"
}

# 禁用密钥
disable_key() {
    local key=$1

    if [ -z "$key" ]; then
        echo -e "${RED}❌ 请提供密钥${NC}"
        return 1
    fi

    echo -e "${YELLOW}禁用密钥：$key${NC}"
    echo ""
    echo -e "${YELLOW}编辑 .env 文件，删除 ALLOWED_KEYS 中包含此密钥的条目${NC}"
    echo ""
    echo "$ nano .env"
    echo "# 删除包含 '$key' 的行"
    echo ""
    echo "$ docker-compose restart moderation"
    echo ""
    echo -e "${GREEN}✓ 已禁用${NC}"
}

# 导出 JSON 格式
export_json() {
    echo -e "${BLUE}API 密钥配置（JSON 格式）${NC}"
    echo ""

    cat > /tmp/keys.json << 'EOFJ'
{
  "keys": [
EOFJ

    if [ -f ".env" ]; then
        first=true
        grep "^ALLOWED_KEYS=" .env | cut -d'=' -f2- | tr ',' '\n' | while read line; do
            if [ -z "$line" ]; then continue; fi

            PROJECT=$(echo "$line" | cut -d'|' -f1)
            KEY=$(echo "$line" | cut -d'|' -f2)
            RATE=$(echo "$line" | cut -d'|' -f3)

            if [ "$first" = true ]; then
                first=false
            else
                echo "," >> /tmp/keys.json
            fi

            cat >> /tmp/keys.json << EOF
    {
      "project_id": "$PROJECT",
      "key": "$KEY",
      "rate_limit_per_minute": $RATE,
      "created_at": "$(date -I)"
    }
EOF
        done
    fi

    cat >> /tmp/keys.json << 'EOFJ'

  ]
}
EOFJ

    cat /tmp/keys.json | jq '.'
    rm /tmp/keys.json
}

# 主逻辑
case "${1:-help}" in
    list)
        list_keys
        ;;
    test)
        test_key "$2"
        ;;
    rate-test)
        rate_test "$2" "$3"
        ;;
    create)
        generate_key "$2" "$3"
        ;;
    rotate)
        rotate_key "$2" "$3"
        ;;
    disable)
        disable_key "$2"
        ;;
    export-json)
        export_json
        ;;
    help|*)
        show_help
        ;;
esac
