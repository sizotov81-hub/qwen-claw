package markdown

import (
	"regexp"
	"strings"
)

// Renderer Markdown рендерер
type Renderer struct {
	// Format формат вывода (telegram/cli/html)
	Format string
}

// NewRenderer создаёт новый рендерер
func NewRenderer(format string) *Renderer {
	return &Renderer{
		Format: format,
	}
}

// Render преобразует Markdown в целевой формат
func (r *Renderer) Render(text string) string {
	result := text

	// Сначала курсив (чтобы * не попало в **)
	result = r.renderItalic(result)

	// Затем жирный текст
	result = r.renderBold(result)

	// Код `code`
	result = r.renderCode(result)

	// Блок кода ```code```
	result = r.renderCodeBlock(result)

	// Заголовки # Header
	result = r.renderHeaders(result)

	// Списки - item или 1. item
	result = r.renderLists(result)

	// Ссылки [text](url)
	result = r.renderLinks(result)

	// Цитаты > text
	result = r.renderQuotes(result)

	return result
}

// renderBold обрабатывает жирный текст
func (r *Renderer) renderBold(text string) string {
	switch r.Format {
	case "telegram":
		// **text** -> *text* (Telegram Markdown)
		re := regexp.MustCompile(`\*\*(.+?)\*\*`)
		text = re.ReplaceAllString(text, "*${1}*")
		// __text__ -> *text*
		re = regexp.MustCompile(`__(.+?)__`)
		text = re.ReplaceAllString(text, "*${1}*")
	case "html":
		// **text** -> <strong>text</strong>
		re := regexp.MustCompile(`\*\*(.+?)\*\*`)
		text = re.ReplaceAllString(text, "<strong>${1}</strong>")
		// __text__ -> <strong>text</strong>
		re = regexp.MustCompile(`__(.+?)__`)
		text = re.ReplaceAllString(text, "<strong>${1}</strong>")
	case "cli":
		// **text** -> text (ANSI bold)
		// Используем placeholder чтобы italic не обрабатывал **
		re := regexp.MustCompile(`\*\*(.+?)\*\*`)
		text = re.ReplaceAllString(text, "**TEMP_BOLD${1}TEMP_BOLD**")
		// __text__ -> text (ANSI bold)
		re = regexp.MustCompile(`__(.+?)__`)
		text = re.ReplaceAllString(text, "**TEMP_BOLD${1}TEMP_BOLD**")
		
		// После italic обрабатываем placeholder'ы
		re = regexp.MustCompile(`\*\*TEMP_BOLD(.+?)TEMP_BOLD\*\*`)
		text = re.ReplaceAllString(text, "\x1b[1m${1}\x1b[0m")
	}
	return text
}

// renderItalic обрабатывает курсив
func (r *Renderer) renderItalic(text string) string {
	switch r.Format {
	case "telegram":
		// *text* -> _text_ (Telegram Markdown)
		// Обрабатываем только * не **
		re := regexp.MustCompile(`\*([^*]+)\*`)
		text = re.ReplaceAllString(text, "_${1}_")
	case "html":
		// *text* -> <em>text</em>
		// Не обрабатываем ** (это для bold)
		re := regexp.MustCompile(`\*\*`)
		text = re.ReplaceAllString(text, "TEMP_STARSTAR")
		re = regexp.MustCompile(`\*([^*]+)\*`)
		text = re.ReplaceAllString(text, "<em>${1}</em>")
		re = regexp.MustCompile(`TEMP_STARSTAR`)
		text = re.ReplaceAllString(text, "**")
	case "cli":
		// *text* -> text (ANSI italic)
		// Не обрабатываем ** (это для bold)
		re := regexp.MustCompile(`\*\*`)
		text = re.ReplaceAllString(text, "TEMP_STARSTAR")
		re = regexp.MustCompile(`\*([^*]+)\*`)
		text = re.ReplaceAllString(text, "\x1b[3m${1}\x1b[0m")
		re = regexp.MustCompile(`TEMP_STARSTAR`)
		text = re.ReplaceAllString(text, "**")
	}
	return text
}

// renderCode обрабатывает inline код
func (r *Renderer) renderCode(text string) string {
	switch r.Format {
	case "telegram":
		// `code` -> `code` (Telegram monospace)
		re := regexp.MustCompile("`([^`]+)`")
		text = re.ReplaceAllString(text, "`$1`")
	case "html":
		// `code` -> <code>code</code>
		re := regexp.MustCompile("`([^`]+)`")
		text = re.ReplaceAllString(text, "<code>$1</code>")
	case "cli":
		// `code` -> code (ANSI color)
		re := regexp.MustCompile("`([^`]+)`")
		text = re.ReplaceAllString(text, "\x1b[36m$1\x1b[0m")
	}
	return text
}

// renderCodeBlock обрабатывает блоки кода
func (r *Renderer) renderCodeBlock(text string) string {
	switch r.Format {
	case "telegram":
		// ```code``` -> ```code``` (Telegram pre)
		re := regexp.MustCompile("```([\\s\\S]*?)```")
		text = re.ReplaceAllString(text, "```\n$1\n```")
	case "html":
		// ```code``` -> <pre><code>code</code></pre>
		re := regexp.MustCompile("```([\\s\\S]*?)```")
		text = re.ReplaceAllString(text, "<pre><code>$1</code></pre>")
	case "cli":
		// ```code``` -> code (ANSI color block)
		re := regexp.MustCompile("```([\\s\\S]*?)```")
		text = re.ReplaceAllString(text, "\n\x1b[48;5;236m$1\x1b[0m\n")
	}
	return text
}

// renderHeaders обрабатывает заголовки
func (r *Renderer) renderHeaders(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		switch r.Format {
		case "telegram":
			if strings.HasPrefix(line, "### ") {
				lines[i] = "*" + strings.TrimPrefix(line, "### ") + "*"
			} else if strings.HasPrefix(line, "## ") {
				lines[i] = "*" + strings.TrimPrefix(line, "## ") + "*"
			} else if strings.HasPrefix(line, "# ") {
				lines[i] = "*" + strings.TrimPrefix(line, "# ") + "*"
			}
		case "html":
			if strings.HasPrefix(line, "### ") {
				lines[i] = "<h3>" + strings.TrimPrefix(line, "### ") + "</h3>"
			} else if strings.HasPrefix(line, "## ") {
				lines[i] = "<h2>" + strings.TrimPrefix(line, "## ") + "</h2>"
			} else if strings.HasPrefix(line, "# ") {
				lines[i] = "<h1>" + strings.TrimPrefix(line, "# ") + "</h1>"
			}
		case "cli":
			if strings.HasPrefix(line, "### ") {
				lines[i] = "\x1b[1;33m" + strings.TrimPrefix(line, "### ") + "\x1b[0m"
			} else if strings.HasPrefix(line, "## ") {
				lines[i] = "\x1b[1;34m" + strings.TrimPrefix(line, "## ") + "\x1b[0m"
			} else if strings.HasPrefix(line, "# ") {
				lines[i] = "\x1b[1;32m" + strings.TrimPrefix(line, "# ") + "\x1b[0m"
			}
		}
	}
	return strings.Join(lines, "\n")
}

// renderLists обрабатывает списки
func (r *Renderer) renderLists(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		switch r.Format {
		case "telegram":
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "• ") {
				lines[i] = "• " + strings.TrimPrefix(strings.TrimPrefix(line, "- "), "• ")
			}
		case "html":
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "• ") {
				lines[i] = "<li>" + strings.TrimPrefix(strings.TrimPrefix(line, "- "), "• ") + "</li>"
			}
		case "cli":
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "• ") {
				lines[i] = "  \x1b[33m•\x1b[0m " + strings.TrimPrefix(strings.TrimPrefix(line, "- "), "• ")
			}
		}
	}
	return strings.Join(lines, "\n")
}

// renderLinks обрабатывает ссылки
func (r *Renderer) renderLinks(text string) string {
	switch r.Format {
	case "telegram":
		// [text](url) -> [text](url)
		re := regexp.MustCompile(`\[(.+?)\]\((.+?)\)`)
		text = re.ReplaceAllString(text, "[$1]($2)")
	case "html":
		// [text](url) -> <a href="url">text</a>
		re := regexp.MustCompile(`\[(.+?)\]\((.+?)\)`)
		text = re.ReplaceAllString(text, `<a href="$2">$1</a>`)
	case "cli":
		// [text](url) -> text (URL)
		re := regexp.MustCompile(`\[(.+?)\]\((.+?)\)`)
		text = re.ReplaceAllString(text, "\x1b[34m$1\x1b[0m ($2)")
	}
	return text
}

// renderQuotes обрабатывает цитаты
func (r *Renderer) renderQuotes(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		switch r.Format {
		case "telegram":
			if strings.HasPrefix(line, "> ") {
				lines[i] = "_" + strings.TrimPrefix(line, "> ") + "_"
			}
		case "html":
			if strings.HasPrefix(line, "> ") {
				lines[i] = "<blockquote>" + strings.TrimPrefix(line, "> ") + "</blockquote>"
			}
		case "cli":
			if strings.HasPrefix(line, "> ") {
				lines[i] = "\x1b[90m| " + strings.TrimPrefix(line, "> ") + "\x1b[0m"
			}
		}
	}
	return strings.Join(lines, "\n")
}

// RenderToTelegram преобразует Markdown в формат Telegram
func RenderToTelegram(text string) string {
	r := NewRenderer("telegram")
	return r.Render(text)
}

// RenderToHTML преобразует Markdown в HTML
func RenderToHTML(text string) string {
	r := NewRenderer("html")
	return r.Render(text)
}

// RenderToCLI преобразует Markdown в ANSI CLI формат
func RenderToCLI(text string) string {
	r := NewRenderer("cli")
	return r.Render(text)
}
