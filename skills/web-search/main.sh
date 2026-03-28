#!/bin/bash
# Web Search Skill для Qwen-Claw
# Поиск информации в интернете через DuckDuckGo

set -e

# Конфигурация по умолчанию
TIMEOUT=${TIMEOUT:-30}
MAX_RESULTS=${MAX_RESULTS:-10}
ENGINE=${ENGINE:-duckduckgo}
CACHE_DIR="${HOME}/.qwen-claw/skills/cache"
CACHE_TTL=${CACHE_TTL:-3600}

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Логирование
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Проверка зависимостей
check_dependencies() {
    local deps=("curl" "jq")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "Зависимость не найдена: $dep"
            echo "Установите: sudo apt install $dep (Ubuntu) или brew install $dep (macOS)"
            exit 1
        fi
    done
}

# Создание директории для кэша
init_cache() {
    if [ ! -d "$CACHE_DIR" ]; then
        mkdir -p "$CACHE_DIR"
        log_info "Создана директория кэша: $CACHE_DIR"
    fi
}

# Генерация ключа кэша
get_cache_key() {
    local query="$1"
    local engine="$2"
    echo -n "${engine}:${query}" | md5sum | cut -d' ' -f1
}

# Проверка кэша
get_from_cache() {
    local cache_key="$1"
    local cache_file="${CACHE_DIR}/${cache_key}.json"
    
    if [ -f "$cache_file" ]; then
        local cache_time=$(stat -f %m "$cache_file" 2>/dev/null || stat -c %Y "$cache_file" 2>/dev/null)
        local current_time=$(date +%s)
        local age=$((current_time - cache_time))
        
        if [ "$age" -lt "$CACHE_TTL" ]; then
            log_info "Найдено в кэше (возраст: ${age}с)"
            cat "$cache_file"
            return 0
        else
            log_info "Кэш устарел (возраст: ${age}с)"
            rm -f "$cache_file"
        fi
    fi
    return 1
}

# Сохранение в кэш
save_to_cache() {
    local cache_key="$1"
    local content="$2"
    local cache_file="${CACHE_DIR}/${cache_key}.json"
    echo "$content" > "$cache_file"
    log_info "Сохранено в кэш: $cache_file"
}

# Поиск через DuckDuckGo
search_duckduckgo() {
    local query="$1"
    local max_results="${2:-10}"
    
    # DuckDuckGo HTML API
    local url="https://html.duckduckgo.com/html/"
    local encoded_query=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$query'))")
    
    log_info "Поиск через DuckDuckGo: $query"
    
    local response=$(curl -s -X POST "$url" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "q=$encoded_query" \
        --max-time "$TIMEOUT" \
        -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
    
    if [ $? -ne 0 ]; then
        log_error "Ошибка при выполнении запроса"
        return 1
    fi
    
    # Парсинг результатов (упрощённый)
    echo "$response" | grep -oP '(?<=<a class="result__a" href=")[^"]*' | head -n "$max_results" | while read -r link; do
        echo "🔗 $link"
    done
}

# Поиск через Google (через html2text)
search_google() {
    local query="$1"
    local max_results="${2:-10}"
    
    log_warning "Google поиск может быть ограничен. Используем html2text подход."
    
    local url="https://www.google.com/search"
    local encoded_query=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$query'))")
    
    log_info "Поиск через Google: $query"
    
    local response=$(curl -s -L "$url?q=$encoded_query" \
        --max-time "$TIMEOUT" \
        -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
    
    if [ $? -ne 0 ]; then
        log_error "Ошибка при выполнении запроса"
        return 1
    fi
    
    # Парсинг результатов
    echo "$response" | grep -oP '(?<=<a href="/url\?q=)[^"]*' | grep -v "^https://www.google.com" | head -n "$max_results" | while read -r link; do
        echo "🔗 $link"
    done
}

# Форматированный вывод результатов
format_results() {
    local results="$1"
    local query="$2"
    
    echo ""
    echo "════════════════════════════════════════"
    echo "  Результаты поиска: $query"
    echo "════════════════════════════════════════"
    echo ""
    
    local count=1
    echo "$results" | while read -r line; do
        if [ -n "$line" ]; then
            printf "%2d. %s\n" "$count" "$line"
            ((count++))
        fi
    done
    
    echo ""
    echo "════════════════════════════════════════"
}

# Основная функция
main() {
    # Проверка аргументов
    if [ $# -lt 1 ]; then
        echo "Использование: $0 <поисковый запрос> [опции]"
        echo ""
        echo "Опции:"
        echo "  --engine=duckduckgo|google   Поисковый движок (по умолчанию: duckduckgo)"
        echo "  --max=10                     Максимум результатов"
        echo "  --nocache                    Не использовать кэш"
        echo ""
        echo "Примеры:"
        echo "  $0 \"Golang microservices\""
        echo "  $0 \"AI frameworks\" --engine=google --max=5"
        exit 1
    fi
    
    # Парсинг аргументов
    local query=""
    local use_cache=true
    
    for arg in "$@"; do
        case $arg in
            --engine=*)
                ENGINE="${arg#*=}"
                ;;
            --max=*)
                MAX_RESULTS="${arg#*=}"
                ;;
            --nocache)
                use_cache=false
                ;;
            *)
                if [ -z "$query" ]; then
                    query="$arg"
                fi
                ;;
        esac
    done
    
    if [ -z "$query" ]; then
        log_error "Поисковый запрос не указан"
        exit 1
    fi
    
    # Проверка зависимостей
    check_dependencies
    
    # Инициализация кэша
    init_cache
    
    # Проверка кэша
    local cache_key=$(get_cache_key "$query" "$ENGINE")
    local cached_result=""
    
    if [ "$use_cache" = true ]; then
        cached_result=$(get_from_cache "$cache_key")
    fi
    
    if [ -n "$cached_result" ]; then
        format_results "$cached_result" "$query"
        exit 0
    fi
    
    # Выполнение поиска
    local results=""
    case $ENGINE in
        duckduckgo)
            results=$(search_duckduckgo "$query" "$MAX_RESULTS")
            ;;
        google)
            results=$(search_google "$query" "$MAX_RESULTS")
            ;;
        *)
            log_error "Неизвестный поисковый движок: $ENGINE"
            echo "Доступные: duckduckgo, google"
            exit 1
            ;;
    esac
    
    if [ -z "$results" ]; then
        log_warning "Ничего не найдено"
        exit 0
    fi
    
    # Сохранение в кэш
    save_to_cache "$cache_key" "$results"
    
    # Вывод результатов
    format_results "$results" "$query"
    
    log_success "Поиск завершён"
}

# Запуск основной функции
main "$@"
