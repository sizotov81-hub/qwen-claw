#!/bin/bash
# Скрипт для запуска всех сервисов Qwen-Claw
# Использует timeout для предотвращения завершения процессов

set -e

cd /home/ss/qwen-claw

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Запуск сервисов Qwen-Claw${NC}"
echo "=============================="
echo ""

# Создаём директорию для логов
mkdir -p logs

# Останавливаем старые процессы
echo -e "${YELLOW}🛑 Остановка старых процессов...${NC}"
pkill -TERM api-gateway 2>/dev/null || true
pkill -TERM session-memory 2>/dev/null || true
pkill -TERM qwen-wrapper 2>/dev/null || true
pkill -TERM llm-proxy 2>/dev/null || true
pkill -TERM tools-executor 2>/dev/null || true
pkill -TERM webui 2>/dev/null || true
sleep 2
echo -e "${GREEN}✅ Старые процессы остановлены${NC}"
echo ""

# Функция запуска сервиса
start_service() {
    local name=$1
    local grpc_port=$2
    local http_port=$3
    local metrics_port=$4
    local extra_args="${@:5}"
    
    echo -e "${BLUE}📦 Запуск $name...${NC}"
    
    # Используем timeout для предотвращения завершения
    timeout 86400 ./bin/$name \
        --grpc-port=$grpc_port \
        --http-port=$http_port \
        --metrics-port=$metrics_port \
        --log-format=console \
        $extra_args \
        > logs/$name.log 2>&1 &
    
    local pid=$!
    echo $pid > logs/$name.pid
    
    sleep 2
    
    # Проверяем что процесс запущен
    if kill -0 $pid 2>/dev/null; then
        echo -e "${GREEN}✅ $name запущен${NC} (PID: $pid, gRPC: $grpc_port, HTTP: $http_port, Metrics: $metrics_port)"
    else
        echo -e "${RED}❌ $name не запустился${NC}"
        tail -10 logs/$name.log
        return 1
    fi
}

# Запуск API Gateway
echo ""
start_service "api-gateway" 55050 58080 59090

# Запуск Session Memory
start_service "session-memory" 55051 58081 59091

# Запуск Qwen Wrapper
start_service "qwen-wrapper" 55052 58082 59092

# Запуск LLM Proxy
start_service "llm-proxy" 55053 58083 59093

# Запуск Tools Executor
start_service "tools-executor" 55054 58084 59094

# Запуск Web UI
echo ""
echo -e "${BLUE}📦 Запуск Web UI...${NC}"
timeout 86400 ./bin/webui --host 0.0.0.0 --port 64656 --dir ./web-new > logs/webui.log 2>&1 &
WEB_PID=$!
echo $WEB_PID > logs/webui.pid
sleep 2

if kill -0 $WEB_PID 2>/dev/null; then
    echo -e "${GREEN}✅ Web UI запущен${NC} (PID: $WEB_PID, Порт: 64656)"
else
    echo -e "${RED}❌ Web UI не запустился${NC}"
    tail -10 logs/webui.log
fi

echo ""
echo "=============================="
echo -e "${GREEN}✅ Запуск завершён!${NC}"
echo ""
echo -e "${BLUE}🌐 Веб-интерфейс:${NC} http://localhost:64656"
echo ""
echo -e "${BLUE}📋 Health endpoints:${NC}"
echo "   API Gateway:     http://localhost:58080/health"
echo "   Session Memory:  http://localhost:58081/health"
echo "   Qwen Wrapper:    http://localhost:58082/health"
echo "   LLM Proxy:       http://localhost:58083/health"
echo "   Tools Executor:  http://localhost:58084/health"
echo ""
echo -e "${BLUE}📊 Metrics endpoints:${NC}"
echo "   API Gateway:     http://localhost:59090/metrics"
echo "   Session Memory:  http://localhost:59091/metrics"
echo "   Qwen Wrapper:    http://localhost:59092/metrics"
echo "   LLM Proxy:       http://localhost:59093/metrics"
echo "   Tools Executor:  http://localhost:59094/metrics"
echo ""
echo -e "${BLUE}📁 Логи:${NC} logs/*.log"
echo ""
echo -e "${YELLOW}🛑 Для остановки: ./stop-all.sh${NC}"
