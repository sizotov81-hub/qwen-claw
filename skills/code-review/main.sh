#!/bin/bash
# Code Review Skill для Qwen-Claw
# Автоматический ревью кода

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Конфигурация
MAX_FILE_SIZE=${MAX_FILE_SIZE:-1048576}
SUPPORTED_EXTENSIONS=(".go" ".py" ".js" ".ts" ".java" ".rs")

# Счётчики
ERRORS=0
WARNINGS=0
SUGGESTIONS=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    ((ERRORS++))
}

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
    ((WARNINGS++))
}

log_suggestion() {
    echo -e "${CYAN}[💡]${NC} $1"
    ((SUGGESTIONS++))
}

# Проверка зависимостей
check_dependencies() {
    local has_issues=false
    
    if ! command -v git &> /dev/null; then
        log_error "git не найден"
        has_issues=true
    fi
    
    # Опциональные зависимости
    if ! command -v gofmt &> /dev/null; then
        log_warning "gofmt не найден (проверка Go кода будет ограничена)"
    fi
    
    if ! command -v python3 &> /dev/null; then
        log_warning "python3 не найден (проверка Python кода будет ограничена)"
    fi
    
    if [ "$has_issues" = true ]; then
        exit 1
    fi
}

# Проверка расширения файла
is_supported_extension() {
    local file="$1"
    local ext="${file##*.}"
    ext=".$ext"
    
    for supported in "${SUPPORTED_EXTENSIONS[@]}"; do
        if [ "$ext" = "$supported" ]; then
            return 0
        fi
    done
    return 1
}

# Проверка размера файла
check_file_size() {
    local file="$1"
    local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
    
    if [ "$size" -gt "$MAX_FILE_SIZE" ]; then
        log_warning "Файл слишком большой: $file ($size байт)"
        return 1
    fi
    return 0
}

# Проверка Go кода
review_go() {
    local file="$1"
    log_info "Анализ Go: $file"
    
    # Проверка через gofmt
    if command -v gofmt &> /dev/null; then
        local fmt_issues=$(gofmt -l "$file" 2>/dev/null)
        if [ -n "$fmt_issues" ]; then
            log_error "gofmt: файл не отформатирован"
        else
            log_success "gofmt: форматирование корректно"
        fi
    fi
    
    # Проверка через go vet (если доступен)
    if command -v go &> /dev/null; then
        local vet_issues=$(go vet "$file" 2>&1)
        if [ -n "$vet_issues" ]; then
            log_error "go vet: $vet_issues"
        fi
    fi
    
    # Проверка на распространённые ошибки
    if grep -q "fmt.Println" "$file"; then
        log_suggestion "Consider using logger instead of fmt.Println"
    fi
    
    if grep -q "panic(" "$file"; then
        log_warning "Found panic() - consider returning error instead"
    fi
    
    if grep -q "TODO\|FIXME\|XXX" "$file"; then
        log_suggestion "Found TODO/FIXME comments"
    fi
}

# Проверка Python кода
review_python() {
    local file="$1"
    log_info "Анализ Python: $file"
    
    # Проверка синтаксиса
    if command -v python3 &> /dev/null; then
        if ! python3 -m py_compile "$file" 2>/dev/null; then
            log_error "Python syntax error"
        else
            log_success "Python syntax: OK"
        fi
    fi
    
    # Проверка через flake8 (если доступен)
    if command -v flake8 &> /dev/null; then
        local flake8_issues=$(flake8 --max-line-length=120 "$file" 2>/dev/null)
        if [ -n "$flake8_issues" ]; then
            log_warning "flake8 issues found"
            echo "$flake8_issues" | head -5
        fi
    fi
    
    # Проверка на распространённые ошибки
    if grep -q "print(" "$file"; then
        log_suggestion "Consider using logging instead of print()"
    fi
    
    if grep -q "import \*" "$file"; then
        log_warning "Avoid wildcard imports (import *)"
    fi
}

# Проверка JavaScript/TypeScript кода
review_js_ts() {
    local file="$1"
    log_info "Анализ JS/TS: $file"
    
    # Проверка через eslint (если доступен)
    if command -v eslint &> /dev/null; then
        local eslint_issues=$(eslint "$file" 2>/dev/null)
        if [ -n "$eslint_issues" ]; then
            log_warning "eslint issues found"
            echo "$eslint_issues" | head -5
        fi
    fi
    
    # Проверка на распространённые ошибки
    if grep -q "console.log" "$file"; then
        log_suggestion "Consider removing console.log in production code"
    fi
    
    if grep -q "var " "$file"; then
        log_suggestion "Prefer let/const over var"
    fi
    
    if grep -q "==" "$file" && ! grep -q "===" "$file"; then
        log_warning "Consider using === instead of =="
    fi
}

# Ревью Git diff
review_git_diff() {
    log_info "Анализ Git diff"
    
    local changed_files=$(git diff --name-only HEAD 2>/dev/null)
    
    if [ -z "$changed_files" ]; then
        log_info "Нет изменений в working directory"
        return
    fi
    
    for file in $changed_files; do
        if [ -f "$file" ] && is_supported_extension "$file" && check_file_size "$file"; then
            case "${file##*.}" in
                go) review_go "$file" ;;
                py) review_python "$file" ;;
                js|ts) review_js_ts "$file" ;;
            esac
        fi
    done
}

# Вывод итогов
print_summary() {
    echo ""
    echo "════════════════════════════════════════"
    echo "  Code Review Summary"
    echo "════════════════════════════════════════"
    echo -e "  ${GREEN}✓ Passed:${NC}   $((ERRORS + WARNINGS + SUGGESTIONS - ERRORS - WARNINGS))"
    echo -e "  ${RED}✗ Errors:${NC}   $ERRORS"
    echo -e "  ${YELLOW}! Warnings:${NC} $WARNINGS"
    echo -e "  ${CYAN}💡 Suggestions:${NC} $SUGGESTIONS"
    echo "════════════════════════════════════════"
    
    if [ "$ERRORS" -gt 0 ]; then
        echo -e "${RED}Review failed with errors${NC}"
        exit 1
    else
        echo -e "${GREEN}Review passed!${NC}"
    fi
}

# Основная функция
main() {
    if [ $# -lt 1 ]; then
        echo "Использование: $0 <file|directory|options>"
        echo ""
        echo "Опции:"
        echo "  --git-diff    Анализировать изменения в Git"
        echo ""
        echo "Примеры:"
        echo "  $0 main.go"
        echo "  $0 ./src/"
        echo "  $0 --git-diff"
        exit 1
    fi
    
    check_dependencies
    
    # Проверка Git diff
    if [ "$1" = "--git-diff" ]; then
        review_git_diff
        print_summary
        exit 0
    fi
    
    local target="$1"
    
    # Если директория
    if [ -d "$target" ]; then
        log_info "Анализ директории: $target"
        
        find "$target" -type f \( -name "*.go" -o -name "*.py" -o -name "*.js" -o -name "*.ts" \) | while read -r file; do
            if check_file_size "$file"; then
                case "${file##*.}" in
                    go) review_go "$file" ;;
                    py) review_python "$file" ;;
                    js|ts) review_js_ts "$file" ;;
                esac
            fi
        done
    # Если файл
    elif [ -f "$target" ]; then
        if ! is_supported_extension "$target"; then
            log_error "Неподдерживаемое расширение: $target"
            echo "Поддерживаются: ${SUPPORTED_EXTENSIONS[*]}"
            exit 1
        fi
        
        if ! check_file_size "$target"; then
            exit 1
        fi
        
        case "${target##*.}" in
            go) review_go "$target" ;;
            py) review_python "$target" ;;
            js|ts) review_js_ts "$target" ;;
        esac
    else
        log_error "Файл или директория не найдены: $target"
        exit 1
    fi
    
    print_summary
}

main "$@"
