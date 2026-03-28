// Qwen-Claw Web UI Client

// Конфигурация
const API_BASE_URL = window.location.origin;
const SERVICES = {
    'api-gateway': { port: 58080, url: '' },
    'session-memory': { port: 58081, url: '' },
    'qwen-wrapper': { port: 58082, url: '' },
    'llm-proxy': { port: 58083, url: '' },
    'tools-executor': { port: 58084, url: '' }
};

// Состояние
let currentSession = null;
let messageHistory = [];
let isSending = false;

// Инициализация
document.addEventListener('DOMContentLoaded', () => {
    console.log('🚀 DOMContentLoaded - начинаем инициализацию...');
    checkServiceHealth();
    loadSessions();
    autoResizeTextarea();
    
    // Periodic health check
    setInterval(checkServiceHealth, 30000);
    
    console.log('✅ Инициализация завершена');
    console.log('sendMessage функция:', typeof sendMessage);
});

// Auto-resize textarea
function autoResizeTextarea() {
    const textarea = document.getElementById('message-input');
    textarea.addEventListener('input', function() {
        this.style.height = 'auto';
        this.style.height = Math.min(this.scrollHeight, 200) + 'px';
    });
}

// Проверка здоровья сервисов
async function checkServiceHealth() {
    // В реальном приложении нужен proxy server для health checks
    // Сейчас используем заглушки так как CORS блокирует запросы
    
    const services = {
        'api-gateway': true,  // Предполагаем что работает
        'session-memory': true,
        'qwen-wrapper': true,
        'llm-proxy': true,
        'tools-executor': true
    };
    
    // Пробуем проверить API Gateway через наш proxy
    try {
        const response = await fetch('/api/health');
        const data = await response.json();
        services['api-gateway'] = data.status === 'ok';
    } catch (e) {
        console.log('Health check error:', e.message);
    }
    
    // Обновляем статусы
    for (const [name, isOnline] of Object.entries(services)) {
        updateServiceStatus(name, isOnline);
    }
}

function updateServiceStatus(name, isOnline) {
    const indicator = document.getElementById(`${name}-status`);
    const badge = document.getElementById(`${name}-badge`);
    
    if (indicator) {
        indicator.className = `status-indicator ${isOnline ? '' : 'offline'}`;
    }
    
    if (badge) {
        badge.className = `status-badge ${isOnline ? 'online' : 'offline'}`;
        badge.textContent = isOnline ? 'Online' : 'Offline';
    }
}

// Загрузка сессий
async function loadSessions() {
    try {
        const response = await fetch(`${API_BASE_URL}/api/sessions`);
        const sessions = await response.json();
        
        const container = document.getElementById('sessions-list');
        container.innerHTML = '';
        
        if (sessions.length === 0) {
            container.innerHTML = '<div style="color:var(--text-secondary);text-align:center;padding:20px;">Нет сессий</div>';
            return;
        }
        
        sessions.forEach(session => {
            const item = document.createElement('div');
            item.className = `session-item ${session.id === currentSession ? 'active' : ''}`;
            item.onclick = () => loadSession(session.id);
            item.innerHTML = `
                <i class="fas fa-comment"></i>
                <span style="flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${session.title || 'Новый чат'}</span>
            `;
            container.appendChild(item);
        });
    } catch (error) {
        console.error('Failed to load sessions:', error);
    }
}

// Создание нового чата
function newChat() {
    currentSession = null;
    messageHistory = [];
    document.getElementById('chat-messages').innerHTML = `
        <div class="message assistant">
            <div class="message-content">
                <p>👋 Привет! Я <strong>Qwen-Claw</strong>, ваш AI-помощник.</p>
                <p style="margin-top: 12px;">Чем могу помочь?</p>
            </div>
        </div>
    `;
    loadSessions();
}

// Загрузка сессии
async function loadSession(sessionId) {
    currentSession = sessionId;
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/sessions/${sessionId}/messages`);
        const messages = await response.json();
        
        messageHistory = messages;
        renderMessages(messages);
        loadSessions();
    } catch (error) {
        console.error('Failed to load session:', error);
    }
}

// Рендеринг сообщений
function renderMessages(messages) {
    const container = document.getElementById('chat-messages');
    container.innerHTML = '';
    
    messages.forEach(msg => {
        appendMessage(msg.role, msg.content);
    });
    
    scrollToBottom();
}

// Отправка сообщения
async function sendMessage() {
    console.log('📤 sendMessage вызван!');
    
    const input = document.getElementById('message-input');
    const message = input.value.trim();
    
    console.log('input element:', input);
    console.log('message:', message, 'isSending:', isSending);
    
    // Alert для отладки
    if (!message) {
        alert('⚠️ Поле ввода пустое! Проверьте что textarea существует.');
        return;
    }
    
    if (isSending) {
        console.log('❌ Отмена: isSending=', isSending);
        return;
    }
    
    isSending = true;
    document.getElementById('send-btn').disabled = true;
    
    // Добавляем сообщение пользователя
    appendMessage('user', message);
    input.value = '';
    input.style.height = 'auto';
    scrollToBottom();
    
    // Показываем индикатор набора
    showTypingIndicator();
    
    try {
        // Пробуем отправить на API Gateway
        const response = await fetch(`${API_BASE_URL}/api/chat`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                session_id: currentSession,
                content: message,
                model: document.getElementById('model-select')?.value || 'gpt-4'
            })
        });
        
        // Удаляем индикатор набора
        removeTypingIndicator();
        
        if (response.ok) {
            const data = await response.json();
            appendMessage('assistant', data.response || data.content);
        } else {
            // API Gateway недоступен - используем заглушку
            appendMessage('assistant', generateMockResponse(message));
        }
    } catch (error) {
        removeTypingIndicator();
        // Используем заглушку при ошибке
        appendMessage('assistant', generateMockResponse(message));
    } finally {
        isSending = false;
        document.getElementById('send-btn').disabled = false;
        scrollToBottom();
    }
}

// Генерация заглушки ответа
function generateMockResponse(message) {
    const responses = [
        `🤖 Я получил ваш запрос: "${message}"\n\n**Примечание:** API Gateway сейчас недоступен. Для полноценной работы убедитесь, что все сервисы запущены.`,
        `👋 Здравствуйте! Ваш запрос: "${message}"\n\nЯ работаю в демо-режиме. Подключите API Gateway для полной функциональности.`,
        `✅ Запрос принят: "${message}"\n\nДля выполнения команды необходим запущенный Qwen Wrapper Service.`
    ];
    return responses[Math.floor(Math.random() * responses.length)];
}

// Добавление сообщения в чат
function appendMessage(role, content) {
    const container = document.getElementById('chat-messages');
    
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${role}`;
    
    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    
    // Рендерим Markdown
    contentDiv.innerHTML = marked.parse(content);
    
    // Подсветка кода
    contentDiv.querySelectorAll('pre code').forEach(block => {
        hljs.highlightElement(block);
    });
    
    messageDiv.appendChild(contentDiv);
    container.appendChild(messageDiv);
    scrollToBottom();
}

// Индикатор набора
function showTypingIndicator() {
    const container = document.getElementById('chat-messages');
    
    const indicator = document.createElement('div');
    indicator.id = 'typing-indicator';
    indicator.className = 'message assistant';
    indicator.innerHTML = `
        <div class="message-content">
            <div class="typing-indicator">
                <div class="typing-dot"></div>
                <div class="typing-dot"></div>
                <div class="typing-dot"></div>
            </div>
        </div>
    `;
    
    container.appendChild(indicator);
    scrollToBottom();
}

function removeTypingIndicator() {
    const indicator = document.getElementById('typing-indicator');
    if (indicator) {
        indicator.remove();
    }
}

// Скролл вниз
function scrollToBottom() {
    const container = document.getElementById('chat-messages');
    container.scrollTop = container.scrollHeight;
}

// Настройки
function openSettings() {
    document.getElementById('settings-modal').classList.add('active');
}

function closeSettings() {
    document.getElementById('settings-modal').classList.remove('active');
}

function saveSettings() {
    const settings = {
        model: document.getElementById('model-select').value,
        approval_mode: document.getElementById('approval-mode').value,
        timeout: parseInt(document.getElementById('timeout-input').value)
    };
    
    localStorage.setItem('qwen-claw-settings', JSON.stringify(settings));
    closeSettings();
    
    // Показываем уведомление
    showNotification('Настройки сохранены', 'success');
}

// Уведомления
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: ${type === 'success' ? 'var(--success)' : 'var(--accent)'};
        color: white;
        padding: 12px 20px;
        border-radius: 8px;
        z-index: 2000;
        animation: slideIn 0.3s ease;
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Загрузка настроек
function loadSettings() {
    const settings = localStorage.getItem('qwen-claw-settings');
    if (settings) {
        const parsed = JSON.parse(settings);
        if (parsed.model) document.getElementById('model-select').value = parsed.model;
        if (parsed.approval_mode) document.getElementById('approval-mode').value = parsed.approval_mode;
        if (parsed.timeout) document.getElementById('timeout-input').value = parsed.timeout;
    }
}

// Экспорт функций для глобального доступа
window.sendMessage = sendMessage;
window.newChat = newChat;
window.loadSession = loadSession;
window.openSettings = openSettings;
window.closeSettings = closeSettings;
window.saveSettings = saveSettings;

// Добавляем кнопку настроек в sidebar
const sidebarHeader = document.querySelector('.sidebar-header');
const settingsBtn = document.createElement('button');
settingsBtn.innerHTML = '<i class="fas fa-cog"></i>';
settingsBtn.style.cssText = `
    position: absolute;
    top: 20px;
    right: 20px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 20px;
`;
settingsBtn.onclick = openSettings;
sidebarHeader.appendChild(settingsBtn);

// Загружаем настройки при старте
loadSettings();
