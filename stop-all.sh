#!/bin/bash
# Скрипт для остановки всех сервисов Qwen-Claw
# Использует Graceful Shutdown (SIGTERM) с таймаутом

set -e

cd /home/ss/qwen-claw

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Таймаут ожидания остановки (секунды)
GRACEFUL_TIMEOUT=${GRACEFUL_TIMEOUT:-30}

echo -e "${BLUE}🛑 Остановка сервисов Qwen-Claw (Graceful Shutdown)${NC}"
echo "============================================="
echo ""

# Список сервисов в обратном порядке (от зависимых к независимым)
SERVICES=(
    "webui:64656"
    "tools-executor:58084"
    "llm-proxy:58083"
    "qwen-wrapper:58082"
    "session-memory:58081"
    "api-gateway:58080"
)

# Функция остановки сервиса
stop_service() {
    local name=$1
    local http_port=$2
    local pid_file="logs/$name.pid"
    
    echo -e "${YELLOW}⏹️  Остановка $name...${NC}"
    
    local pid=""
    
    # Сначала пытаемся найти процесс по имени
    pid=$(pgrep -f "bin/$name" 2>/dev/null | head -1)
    
    # Если не нашли, пробуем получить PID из файла
    if [ -z "$pid" ] && [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
    fi
    
    # Останавливаем процесс
    if [ -n "$pid" ] && kill -0 $pid 2>/dev/null; then
        # Отправляем SIGTERM для graceful shutdown
        kill -TERM $pid 2>/dev/null || true
        echo -e "   📭 Отправлен SIGTERM (PID: $pid)"
        
        # Ждём завершения процесса
        local count=0
        while kill -0 $pid 2>/dev/null && [ $count -lt $GRACEFUL_TIMEOUT ]; do
            sleep 1
            count=$((count + 1))
            if [ $((count % 5)) -eq 0 ]; then
                echo -e "   ⏳ Ожидание завершения... ($count сек)"
            fi
        done
        
        # Если процесс всё ещё работает - отправляем SIGKILL
        if kill -0 $pid 2>/dev/null; then
            echo -e "   ${YELLOW}⚠️  Graceful shutdown не удался, отправляем SIGKILL${NC}"
            kill -9 $pid 2>/dev/null || true
            sleep 1
        else
            echo -e "${GREEN}   ✅ Остановлен корректно${NC}"
        fi
    else
        echo -e "   ${BLUE}ℹ️  Процесс не запущен${NC}"
    fi
    
    # Удаляем PID файл
    rm -f "$pid_file"
    
    echo ""
}

# Останавливаем сервисы
for service in "${SERVICES[@]}"; do
    name="${service%%:*}"
    port="${service##*:}"
    stop_service "$name" "$port"
done

# Проверяем что все процессы остановлены
echo -e "${BLUE}🔍 Проверка остановки...${NC}"
sleep 2

running=0
for service in "${SERVICES[@]}"; do
    name="${service%%:*}"
    if pgrep -f "bin/$name" > /dev/null 2>&1; then
        echo -e "${RED}❌ $name всё ещё работает${NC}"
        running=$((running + 1))
    fi
done

if [ $running -eq 0 ]; then
    echo -e "${GREEN}✅ Все сервисы остановлены${NC}"
else
    echo -e "${YELLOW}⚠️  Некоторые сервисы не остановились. Принудительная остановка...${NC}"
    pkill -9 -f "bin/(api-gateway|session-memory|qwen-wrapper|llm-proxy|tools-executor|webui)" 2>/dev/null || true
    sleep 1
    echo -e "${GREEN}✅ Все сервисы остановлены принудительно${NC}"
fi

echo ""
echo "============================================="
echo -e "${GREEN}🛑 Остановка завершена${NC}"
echo ""

# Показываем статистику
echo -e "${BLUE}📊 Статистика:${NC}"
echo "   Остановлено сервисов: ${#SERVICES[@]}"
echo "   Таймаут: $GRACEFUL_TIMEOUT сек"
echo ""

# Показываем последние логи
echo -e "${BLUE}📁 Последние логи:${NC}"
for service in "${SERVICES[@]}"; do
    name="${service%%:*}"
    log_file="logs/$name.log"
    if [ -f "$log_file" ]; then
        last_line=$(tail -1 "$log_file" 2>/dev/null || echo "N/A")
        echo "   $name: ${last_line:0:80}..."
    fi
done
echo ""
