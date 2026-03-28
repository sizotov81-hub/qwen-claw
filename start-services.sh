#!/bin/bash
# Скрипт для запуска микросервисов Qwen-Claw

set -e

echo "🚀 Запуск микросервисов Qwen-Claw"
echo "=================================="
echo ""

BIN_DIR="./bin"
LOG_DIR="./logs"

# Создаём директорию для логов
mkdir -p "$LOG_DIR"

# Функция для запуска сервиса
start_service() {
    local name=$1
    local grpc_port=$2
    local http_port=$3
    local metrics_port=$4
    
    echo "📦 Запуск $name..."
    
    $BIN_DIR/$name \
        --grpc-port=$grpc_port \
        --http-port=$http_port \
        --metrics-port=$metrics_port \
        --log-format=console \
        > "$LOG_DIR/$name.log" 2>&1 &
    
    echo $! > "$LOG_DIR/$name.pid"
    
    sleep 1
    
    if kill -0 $(cat "$LOG_DIR/$name.pid") 2>/dev/null; then
        echo "✅ $name запущен (gRPC: $grpc_port, HTTP: $http_port, Metrics: $metrics_port)"
    else
        echo "❌ $name не запустился"
        exit 1
    fi
}

# Функция для остановки сервиса
stop_service() {
    local name=$1
    
    if [ -f "$LOG_DIR/$name.pid" ]; then
        pid=$(cat "$LOG_DIR/$name.pid")
        if kill -0 $pid 2>/dev/null; then
            kill $pid
            echo "✅ $name остановлен"
        else
            echo "⚠️  $name не запущен"
        fi
        rm -f "$LOG_DIR/$name.pid"
    fi
}

# Обработка команд
case "${1:-start}" in
    start)
        echo "Запуск всех сервисов..."
        echo ""
        
        start_service "api-gateway" 55050 58080 59090
        start_service "session-memory" 55051 58081 59091
        start_service "qwen-wrapper" 55052 58082 59092
        start_service "llm-proxy" 55053 58083 59093
        start_service "tools-executor" 55054 58084 59094

        echo ""
        echo "=================================="
        echo "✅ Все сервисы запущены!"
        echo ""
        echo "Порты:"
        echo "  API Gateway:     gRPC 55050, HTTP 58080"
        echo "  Session Memory:  gRPC 55051, HTTP 58081"
        echo "  Qwen Wrapper:    gRPC 55052, HTTP 58082"
        echo "  LLM Proxy:       gRPC 55053, HTTP 58083"
        echo "  Tools Executor:  gRPC 55054, HTTP 58084"
        echo ""
        echo "Health endpoints:"
        echo "  http://localhost:58080/health"
        echo "  http://localhost:58081/health"
        echo "  http://localhost:58082/health"
        echo "  http://localhost:58083/health"
        echo "  http://localhost:58084/health"
        echo ""
        echo "Metrics endpoints:"
        echo "  http://localhost:59090/metrics"
        echo "  http://localhost:59091/metrics"
        echo "  http://localhost:59092/metrics"
        echo "  http://localhost:59093/metrics"
        echo "  http://localhost:59094/metrics"
        echo ""
        echo "Логи: $LOG_DIR/*.log"
        echo ""
        echo "Для остановки: $0 stop"
        ;;
    
    stop)
        echo "Остановка всех сервисов..."
        echo ""
        
        stop_service "tools-executor"
        stop_service "llm-proxy"
        stop_service "qwen-wrapper"
        stop_service "session-memory"
        stop_service "api-gateway"
        
        echo ""
        echo "✅ Все сервисы остановлены"
        ;;
    
    status)
        echo "Статус сервисов:"
        echo ""
        
        for service in api-gateway session-memory qwen-wrapper llm-proxy tools-executor; do
            if [ -f "$LOG_DIR/$service.pid" ]; then
                pid=$(cat "$LOG_DIR/$service.pid")
                if kill -0 $pid 2>/dev/null; then
                    echo "✅ $service запущен (PID: $pid)"
                else
                    echo "❌ $service не работает (PID: $pid)"
                fi
            else
                echo "⚠️  $service не запущен"
            fi
        done
        ;;
    
    restart)
        $0 stop
        sleep 2
        $0 start
        ;;
    
    *)
        echo "Использование: $0 {start|stop|status|restart}"
        exit 1
        ;;
esac
