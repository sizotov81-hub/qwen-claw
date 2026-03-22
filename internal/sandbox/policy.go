package sandbox

import (
	"fmt"
	"regexp"
	"strings"
)

// SecurityPolicy политика безопасности sandbox
type SecurityPolicy struct {
	// BlockedCommands заблокированные команды
	BlockedCommands []string
	
	// BlockedPatterns заблокированные паттерны (regex)
	BlockedPatterns []*regexp.Regexp
	
	// AllowedPaths разрешённые пути
	AllowedPaths []string
	
	// BlockedPaths заблокированные пути
	BlockedPaths []string
	
	// MaxCommandLength максимальная длина команды
	MaxCommandLength int
	
	// RequireSandbox требует sandbox для команды
	RequireSandbox map[string]bool
}

// DefaultSecurityPolicy возвращает политику по умолчанию
func DefaultSecurityPolicy() *SecurityPolicy {
	blockedPatterns := []string{
		`rm\s+(-[rf]+\s+)?/`,         // rm -rf /
		`mkfs`,                        // Форматирование
		`dd\s+.*of=/dev/`,             // dd в устройство
		`:\(\)\{\s*:\|:&\s*\};:`,      // Fork bomb
		`chmod\s+(-)?777`,             // chmod 777
		`curl.*\|\s*(ba)?sh`,          // curl | sh
		`wget.*\|\s*(ba)?sh`,          // wget | sh
		`/etc/passwd`,                 // Доступ к passwd
		`/etc/shadow`,                 // Доступ к shadow
	}
	
	compiledPatterns := make([]*regexp.Regexp, 0, len(blockedPatterns))
	for _, pattern := range blockedPatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			compiledPatterns = append(compiledPatterns, re)
		}
	}
	
	return &SecurityPolicy{
		BlockedCommands: []string{
			"rm",
			"mkfs",
			"dd",
			"shutdown",
			"reboot",
			"halt",
			"poweroff",
			"kill",
			"pkill",
			"killall",
			"su",
			"sudo",
			"visudo",
			"passwd",
			"useradd",
			"userdel",
			"usermod",
			"groupadd",
			"groupdel",
			"mount",
			"umount",
			"fdisk",
			"parted",
			"iptables",
			"firewall-cmd",
			"systemctl",
			"service",
			"crontab",
			"at",
		},
		BlockedPatterns: compiledPatterns,
		AllowedPaths: []string{
			"/workspace",
			"/tmp",
			"/home",
		},
		BlockedPaths: []string{
			"/",
			"/etc",
			"/root",
			"/var",
			"/usr",
			"/bin",
			"/sbin",
			"/boot",
			"/dev",
			"/proc",
			"/sys",
		},
		MaxCommandLength: 10000,
		RequireSandbox: map[string]bool{
			"bash":  true,
			"sh":    true,
			"zsh":   true,
			"fish":  true,
			"python": true,
			"node":  true,
			"ruby":  true,
			"perl":  true,
			"php":   true,
		},
	}
}

// CheckCommand проверяет команду на безопасность
func (p *SecurityPolicy) CheckCommand(command string) error {
	// Проверяем длину
	if len(command) > p.MaxCommandLength {
		return ErrSecurityViolation{
			Reason:  "command_too_long",
			Details: fmt.Sprintf("command length %d exceeds limit %d", len(command), p.MaxCommandLength),
		}
	}
	
	// Проверяем заблокированные команды
	cmd := strings.Fields(command)[0]
	for _, blocked := range p.BlockedCommands {
		if cmd == blocked {
			return ErrSecurityViolation{
				Reason:  "blocked_command",
				Details: fmt.Sprintf("command '%s' is blocked", blocked),
			}
		}
	}
	
	// Проверяем паттерны
	for _, pattern := range p.BlockedPatterns {
		if pattern.MatchString(command) {
			return ErrSecurityViolation{
				Reason:  "blocked_pattern",
				Details: fmt.Sprintf("command matches blocked pattern '%s'", pattern.String()),
			}
		}
	}
	
	// Проверяем пути
	for _, blocked := range p.BlockedPaths {
		if strings.Contains(command, blocked) {
			// Исключаем разрешённые подпути
			allowed := false
			for _, allowedPath := range p.AllowedPaths {
				if strings.HasPrefix(blocked, allowedPath) {
					allowed = true
					break
				}
			}
			if !allowed {
				return ErrSecurityViolation{
					Reason:  "blocked_path",
					Details: fmt.Sprintf("access to '%s' is blocked", blocked),
				}
			}
		}
	}
	
	return nil
}

// RequiresSandbox проверяет, требует ли команда sandbox
func (p *SecurityPolicy) RequiresSandbox(command string) bool {
	cmd := strings.Fields(command)[0]
	return p.RequireSandbox[cmd]
}

// ErrSecurityViolation ошибка нарушения безопасности
type ErrSecurityViolation struct {
	Reason  string
	Details string
}

func (e ErrSecurityViolation) Error() string {
	return fmt.Sprintf("security violation: %s (%s)", e.Reason, e.Details)
}

// SecurityLevel уровень безопасности
type SecurityLevel string

const (
	// SecurityLevelLow низкий уровень (sandbox отключён)
	SecurityLevelLow SecurityLevel = "low"
	
	// SecurityLevelMedium средний уровень (базовые проверки)
	SecurityLevelMedium SecurityLevel = "medium"
	
	// SecurityLevelHigh высокий уровень (строгие проверки)
	SecurityLevelHigh SecurityLevel = "high"
	
	// SecurityLevelMaximum максимальный уровень (полная изоляция)
	SecurityLevelMaximum SecurityLevel = "maximum"
)

// GetSecurityPolicyForLevel возвращает политику для уровня
func GetSecurityPolicyForLevel(level SecurityLevel) *SecurityPolicy {
	switch level {
	case SecurityLevelLow:
		// Минимальные ограничения
		return &SecurityPolicy{
			BlockedCommands: []string{
				"rm", "mkfs", "dd", "shutdown", "reboot",
			},
			MaxCommandLength: 50000,
		}
		
	case SecurityLevelMedium:
		// Базовые ограничения
		return DefaultSecurityPolicy()
		
	case SecurityLevelHigh:
		// Строгие ограничения
		policy := DefaultSecurityPolicy()
		policy.BlockedCommands = append(policy.BlockedCommands,
			"curl", "wget", "git", "npm", "pip",
		)
		policy.AllowedPaths = []string{"/workspace"}
		policy.BlockedPaths = append(policy.BlockedPaths, "/home")
		return policy
		
	case SecurityLevelMaximum:
		// Максимальные ограничения
		policy := DefaultSecurityPolicy()
		policy.BlockedCommands = []string{"*"} // Блокировать всё
		policy.AllowedPaths = []string{"/workspace"}
		policy.MaxCommandLength = 1000
		return policy
		
	default:
		return DefaultSecurityPolicy()
	}
}

// AuditResult результат аудита безопасности
type AuditResult struct {
	// Safe безопасна ли команда
	Safe bool
	
	// Warnings предупреждения
	Warnings []string
	
	// Violations нарушения
	Violations []ErrSecurityViolation
	
	// Recommendation рекомендация
	Recommendation string
}

// AuditCommand аудирует команду
func (p *SecurityPolicy) AuditCommand(command string) *AuditResult {
	result := &AuditResult{
		Safe:       true,
		Warnings:   make([]string, 0),
		Violations: make([]ErrSecurityViolation, 0),
	}
	
	// Проверяем длину
	if len(command) > p.MaxCommandLength {
		result.Safe = false
		result.Violations = append(result.Violations, ErrSecurityViolation{
			Reason:  "command_too_long",
			Details: fmt.Sprintf("length %d > %d", len(command), p.MaxCommandLength),
		})
	} else if len(command) > p.MaxCommandLength/2 {
		result.Warnings = append(result.Warnings, "command is very long")
	}
	
	// Проверяем команды
	cmd := strings.Fields(command)[0]
	for _, blocked := range p.BlockedCommands {
		if cmd == blocked {
			result.Safe = false
			result.Violations = append(result.Violations, ErrSecurityViolation{
				Reason:  "blocked_command",
				Details: fmt.Sprintf("'%s' is blocked", blocked),
			})
		}
	}
	
	// Проверяем паттерны
	for _, pattern := range p.BlockedPatterns {
		if pattern.MatchString(command) {
			result.Safe = false
			result.Violations = append(result.Violations, ErrSecurityViolation{
				Reason:  "blocked_pattern",
				Details: fmt.Sprintf("matches '%s'", pattern.String()),
			})
		}
	}
	
	// Формируем рекомендацию
	if result.Safe {
		if p.RequiresSandbox(command) {
			result.Recommendation = "run_in_sandbox"
		} else {
			result.Recommendation = "safe_to_run"
		}
	} else {
		result.Recommendation = "block_command"
	}
	
	return result
}
