#!/bin/bash
# GitHub Tools Skill для Qwen-Claw
# Работа с GitHub API

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Конфигурация
GITHUB_API="${GITHUB_API_URL:-https://api.github.com}"
TIMEOUT=${TIMEOUT:-30}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# Проверка токена
check_token() {
    if [ -z "$GITHUB_TOKEN" ]; then
        log_error "GITHUB_TOKEN не установлен"
        echo "Установите: export GITHUB_TOKEN=ghp_xxxxxxxxxxxx"
        exit 1
    fi
}

# Проверка зависимостей
check_dependencies() {
    local deps=("curl" "jq")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "Зависимость не найдена: $dep"
            exit 1
        fi
    done
}

# GitHub API запрос
gh_api() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    local url="${GITHUB_API}${endpoint}"
    
    if [ "$method" = "GET" ]; then
        curl -s -L \
            -H "Authorization: token $GITHUB_TOKEN" \
            -H "Accept: application/vnd.github.v3+json" \
            "$url" \
            --max-time "$TIMEOUT"
    elif [ "$method" = "POST" ]; then
        curl -s -L \
            -X POST \
            -H "Authorization: token $GITHUB_TOKEN" \
            -H "Accept: application/vnd.github.v3+json" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$url" \
            --max-time "$TIMEOUT"
    fi
}

# Информация о репозитории
repo_info() {
    local repo="$1"
    
    log_info "Получение информации о репозитории: $repo"
    
    local response=$(gh_api GET "/repos/$repo")
    
    if echo "$response" | jq -e '.message == "Not Found"' > /dev/null 2>&1; then
        log_error "Репозиторий не найден: $repo"
        exit 1
    fi
    
    local name=$(echo "$response" | jq -r '.full_name')
    local description=$(echo "$response" | jq -r '.description')
    local stars=$(echo "$response" | jq -r '.stargazers_count')
    local forks=$(echo "$response" | jq -r '.forks_count')
    local language=$(echo "$response" | jq -r '.language')
    local license=$(echo "$response" | jq -r '.license.spdx_id')
    local updated=$(echo "$response" | jq -r '.updated_at')
    
    echo ""
    echo "════════════════════════════════════════"
    echo "  Repository: $name"
    echo "════════════════════════════════════════"
    echo "  Описание:     ${description:-N/A}"
    echo "  Звёзды:       ⭐ $stars"
    echo "  Форки:        🍴 $forks"
    echo "  Язык:         $language"
    echo "  Лицензия:     $license"
    echo "  Обновлено:    $updated"
    echo "════════════════════════════════════════"
}

# Список issues
list_issues() {
    local repo="$1"
    local state="${2:-open}"
    
    log_info "Получение issues для: $repo (state: $state)"
    
    local response=$(gh_api GET "/repos/$repo/issues?state=$state&per_page=20")
    
    local count=$(echo "$response" | jq 'length')
    
    echo ""
    echo "════════════════════════════════════════"
    echo "  Issues ($count found)"
    echo "════════════════════════════════════════"
    
    echo "$response" | jq -r '.[] | "#\(.number) - \(.title) by \(.user.login) [\(.state)]"'
    
    echo "════════════════════════════════════════"
}

# Создание Pull Request
create_pr() {
    local repo="$1"
    local head="$2"
    local base="$3"
    local title="$4"
    local body="${5:-}"
    
    log_info "Создание Pull Request: $repo"
    log_info "  head: $head"
    log_info "  base: $base"
    log_info "  title: $title"
    
    local data=$(jq -n \
        --arg head "$head" \
        --arg base "$base" \
        --arg title "$title" \
        --arg body "$body" \
        '{head: $head, base: $base, title: $title, body: $body}')
    
    local response=$(gh_api POST "/repos/$repo/pulls" "$data")
    
    if echo "$response" | jq -e '.message' > /dev/null 2>&1; then
        local error=$(echo "$response" | jq -r '.message')
        log_error "Ошибка: $error"
        exit 1
    fi
    
    local pr_number=$(echo "$response" | jq -r '.number')
    local pr_url=$(echo "$response" | jq -r '.html_url')
    
    echo ""
    echo "════════════════════════════════════════"
    echo -e "  ${GREEN}✓ Pull Request создан!${NC}"
    echo "════════════════════════════════════════"
    echo "  Номер:  #$pr_number"
    echo "  URL:    $pr_url"
    echo "════════════════════════════════════════"
}

# Список Pull Requests
list_prs() {
    local repo="$1"
    local state="${2:-open}"
    
    log_info "Получение Pull Requests для: $repo (state: $state)"
    
    local response=$(gh_api GET "/repos/$repo/pulls?state=$state&per_page=20")
    
    local count=$(echo "$response" | jq 'length')
    
    echo ""
    echo "════════════════════════════════════════"
    echo "  Pull Requests ($count found)"
    echo "════════════════════════════════════════"
    
    echo "$response" | jq -r '.[] | "#\(.number) - \(.title) by \(.user.login) [\(.state)]"'
    
    echo "════════════════════════════════════════"
}

# Вывод помощи
print_help() {
    echo "GitHub Tools для Qwen-Claw"
    echo ""
    echo "Использование:"
    echo "  $0 <command> [arguments]"
    echo ""
    echo "Команды:"
    echo "  repo <owner/repo>           Информация о репозитории"
    echo "  issues <owner/repo> [state] Список issues (open/closed)"
    echo "  prs <owner/repo> [state]    Список Pull Requests"
    echo "  create-pr <repo> <head> <base> <title> [body]"
    echo "                              Создать Pull Request"
    echo ""
    echo "Примеры:"
    echo "  $0 repo sizotov81-hub/qwen-claw"
    echo "  $0 issues sizotov81-hub/qwen-claw open"
    echo "  $0 create-pr sizotov81-hub/qwen-claw feature-branch main \"Fix bug\""
    echo ""
    echo "Требуется:"
    echo "  export GITHUB_TOKEN=ghp_xxxxxxxxxxxx"
}

# Основная функция
main() {
    if [ $# -lt 1 ]; then
        print_help
        exit 0
    fi
    
    check_dependencies
    check_token
    
    local command="$1"
    shift
    
    case $command in
        repo)
            if [ $# -lt 1 ]; then
                log_error "Укажите репозиторий: owner/repo"
                exit 1
            fi
            repo_info "$1"
            ;;
        
        issues)
            if [ $# -lt 1 ]; then
                log_error "Укажите репозиторий: owner/repo"
                exit 1
            fi
            list_issues "$1" "${2:-open}"
            ;;
        
        prs)
            if [ $# -lt 1 ]; then
                log_error "Укажите репозиторий: owner/repo"
                exit 1
            fi
            list_prs "$1" "${2:-open}"
            ;;
        
        create-pr)
            if [ $# -lt 4 ]; then
                log_error "Недостаточно аргументов"
                echo "Использование: $0 create-pr <repo> <head> <base> <title> [body]"
                exit 1
            fi
            create_pr "$1" "$2" "$3" "$4" "${5:-}"
            ;;
        
        help|--help|-h)
            print_help
            ;;
        
        *)
            log_error "Неизвестная команда: $command"
            print_help
            exit 1
            ;;
    esac
}

main "$@"
