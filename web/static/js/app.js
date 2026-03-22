// Qwen-Claw Web Control UI - Client Application

class QwenClawApp {
    constructor() {
        this.currentTab = 'chat';
        this.ws = null;
        this.sessionId = null;
        this.config = {};
        this.init();
    }

    init() {
        this.setupTabs();
        this.setupChat();
        this.setupWebSocket();
        this.loadSessions();
        this.loadSkills();
        this.loadConfig();
        this.setupEventListeners();
    }

    // Tabs
    setupTabs() {
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                const tab = item.dataset.tab;
                this.switchTab(tab);
            });
        });
    }

    switchTab(tab) {
        this.currentTab = tab;
        
        // Update nav
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
            if (item.dataset.tab === tab) {
                item.classList.add('active');
            }
        });
        
        // Update content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`${tab}-tab`).classList.add('active');
        
        // Load tab-specific data
        this.loadTabData(tab);
    }

    loadTabData(tab) {
        switch(tab) {
            case 'sessions':
                this.loadSessions();
                break;
            case 'skills':
                this.loadSkills();
                break;
            case 'config':
                this.loadConfig();
                break;
            case 'logs':
                this.startLogsStream();
                break;
        }
    }

    // WebSocket
    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            this.updateGatewayStatus(true);
            this.authenticate();
        };
        
        this.ws.onclose = () => {
            this.updateGatewayStatus(false);
            setTimeout(() => this.setupWebSocket(), 3000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showToast('Ошибка подключения к Gateway', 'error');
        };
        
        this.ws.onmessage = (event) => {
            this.handleWebSocketMessage(JSON.parse(event.data));
        };
    }

    authenticate() {
        // В реальной реализации нужен токен
        this.send({
            type: 'auth',
            payload: {
                client_id: 'web_client',
                client_type: 'web',
                token: 'web_token'
            }
        });
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        }
    }

    handleWebSocketMessage(msg) {
        switch(msg.type) {
            case 'auth_response':
                if (msg.payload.success) {
                    this.sessionId = msg.payload.session_id;
                    console.log('Authenticated, session:', this.sessionId);
                }
                break;
            case 'chat_response':
                this.appendMessage('assistant', msg.payload.content);
                this.setSendButtonEnabled(true);
                break;
            case 'error':
                this.showToast(msg.payload.message, 'error');
                this.setSendButtonEnabled(true);
                break;
            case 'event':
                this.handleEvent(msg.payload);
                break;
        }
    }

    handleEvent(event) {
        console.log('Event received:', event);
        
        switch(event.event) {
            case 'client_connected':
            case 'client_disconnected':
                this.loadSessions();
                break;
        }
    }

    updateGatewayStatus(connected) {
        const statusEl = document.getElementById('gateway-status');
        const indicator = document.querySelector('.status-indicator');
        
        if (connected) {
            statusEl.textContent = 'Gateway: подключено';
            indicator.classList.add('online');
            indicator.classList.remove('offline');
        } else {
            statusEl.textContent = 'Gateway: отключено';
            indicator.classList.add('offline');
            indicator.classList.remove('online');
        }
    }

    // Chat
    setupChat() {
        const sendBtn = document.getElementById('send-btn');
        const chatInput = document.getElementById('chat-input');
        
        sendBtn.addEventListener('click', () => this.sendMessage());
        
        chatInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });
    }

    sendMessage() {
        const input = document.getElementById('chat-input');
        const message = input.value.trim();
        
        if (!message) return;
        
        // Append user message
        this.appendMessage('user', message);
        input.value = '';
        
        // Send to server
        this.setSendButtonEnabled(false);
        this.send({
            type: 'chat',
            session_id: this.sessionId,
            payload: {
                content: message,
                model: document.getElementById('model-select').value
            }
        });
        
        // Show typing indicator
        this.showTypingIndicator();
    }

    appendMessage(role, content) {
        const messagesContainer = document.getElementById('chat-messages');
        const messageEl = document.createElement('div');
        messageEl.className = `message ${role}`;
        
        // Simple markdown-like formatting
        const formattedContent = this.formatMessage(content);
        
        messageEl.innerHTML = `
            <div class="message-content">
                <p>${formattedContent}</p>
            </div>
        `;
        
        messagesContainer.appendChild(messageEl);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    formatMessage(content) {
        return content
            .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
            .replace(/`([^`]+)`/g, '<code>$1</code>')
            .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
            .replace(/\*([^*]+)\*/g, '<em>$1</em>')
            .replace(/\n/g, '<br>');
    }

    showTypingIndicator() {
        const messagesContainer = document.getElementById('chat-messages');
        const indicatorEl = document.createElement('div');
        indicatorEl.className = 'message assistant typing-indicator';
        indicatorEl.id = 'typing-indicator';
        indicatorEl.innerHTML = `
            <div class="message-content">
                <p>⏳ Думаю...</p>
            </div>
        `;
        messagesContainer.appendChild(indicatorEl);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    removeTypingIndicator() {
        const indicator = document.getElementById('typing-indicator');
        if (indicator) {
            indicator.remove();
        }
    }

    setSendButtonEnabled(enabled) {
        const btn = document.getElementById('send-btn');
        const spinner = btn.querySelector('.spinner');
        const text = btn.querySelector('span:not(.spinner)');
        
        if (enabled) {
            btn.disabled = false;
            spinner.classList.add('hide');
            text.textContent = 'Отправить';
            this.removeTypingIndicator();
        } else {
            btn.disabled = true;
            spinner.classList.remove('hide');
            text.textContent = 'Отправка...';
        }
    }

    setupEventListeners() {
        document.getElementById('clear-chat').addEventListener('click', () => {
            document.getElementById('chat-messages').innerHTML = '';
            this.appendMessage('assistant', '👋 Чат очищен. Чем могу помочь?');
        });
        
        document.getElementById('new-session-btn').addEventListener('click', () => {
            this.createNewSession();
        });
        
        document.getElementById('save-config').addEventListener('click', () => {
            this.saveConfig();
        });
        
        document.getElementById('refresh-skills').addEventListener('click', () => {
            this.loadSkills();
        });
        
        document.getElementById('skills-search').addEventListener('input', (e) => {
            this.filterSkills(e.target.value);
        });
    }

    // Sessions
    async loadSessions() {
        try {
            const response = await fetch('/api/v1/sessions');
            const data = await response.json();
            
            const grid = document.getElementById('sessions-grid');
            grid.innerHTML = '';
            
            if (data.sessions.length === 0) {
                grid.innerHTML = '<p class="empty-state">Нет активных сессий</p>';
                return;
            }
            
            data.sessions.forEach(session => {
                const card = this.createSessionCard(session);
                grid.appendChild(card);
            });
        } catch (error) {
            console.error('Failed to load sessions:', error);
        }
    }

    createSessionCard(session) {
        const card = document.createElement('div');
        card.className = 'session-card';
        
        const statusClass = session.status === 'active' ? 'active' : 'idle';
        const lastAccess = new Date(session.last_access).toLocaleString();
        
        card.innerHTML = `
            <div class="session-card-header">
                <span class="session-card-title">${session.id}</span>
                <span class="session-card-badge ${statusClass}">${session.status}</span>
            </div>
            <div class="session-card-info">
                <div>Тип: ${session.type}</div>
                <div>Канал: ${session.channel}</div>
                <div>Сообщений: ${session.message_count}</div>
                <div>Последний доступ: ${lastAccess}</div>
            </div>
            <div class="session-card-actions">
                <button class="btn btn-secondary" onclick="app.viewSession('${session.id}')">👁️ Просмотр</button>
                <button class="btn btn-danger" onclick="app.deleteSession('${session.id}')">🗑️ Удалить</button>
            </div>
        `;
        
        return card;
    }

    createNewSession() {
        // В реальной реализации API для создания сессии
        this.showToast('Новая сессия создана', 'success');
        this.loadSessions();
    }

    viewSession(sessionId) {
        // Показать детали сессии в модалке
        document.getElementById('modal-title').textContent = `Сессия: ${sessionId}`;
        document.getElementById('modal-body').innerHTML = `
            <p>Загрузка информации о сессии...</p>
        `;
        openModal();
    }

    deleteSession(sessionId) {
        if (confirm(`Удалить сессию ${sessionId}?`)) {
            // В реальной реализации API для удаления
            this.showToast(`Сессия ${sessionId} удалена`, 'success');
            this.loadSessions();
        }
    }

    // Skills
    async loadSkills() {
        try {
            const response = await fetch('/api/v1/skills');
            const data = await response.json();
            
            const grid = document.getElementById('skills-grid');
            grid.innerHTML = '';
            
            if (data.skills.length === 0) {
                grid.innerHTML = '<p class="empty-state">Нет установленных навыков</p>';
                return;
            }
            
            data.skills.forEach(skill => {
                const card = this.createSkillCard(skill);
                grid.appendChild(card);
            });
        } catch (error) {
            console.error('Failed to load skills:', error);
        }
    }

    createSkillCard(skill) {
        const card = document.createElement('div');
        card.className = 'skill-card';
        
        const statusClass = skill.enabled ? 'enabled' : 'disabled';
        
        card.innerHTML = `
            <div class="skill-card-header">
                <span class="skill-card-title">${skill.name}</span>
                <span class="skill-card-status ${statusClass}"></span>
            </div>
            <p class="skill-card-description">${skill.description}</p>
            ${skill.commands.length > 0 ? `
                <div class="skill-card-commands">
                    <strong>Команды:</strong> ${skill.commands.join(', ')}
                </div>
            ` : ''}
            <div class="skill-card-actions">
                ${skill.enabled ? 
                    `<button class="btn btn-secondary" onclick="app.toggleSkill('${skill.name}', false)">⏸️ Отключить</button>` :
                    `<button class="btn btn-primary" onclick="app.toggleSkill('${skill.name}', true)">▶️ Включить</button>`
                }
                <button class="btn btn-danger" onclick="app.uninstallSkill('${skill.name}')">🗑️ Удалить</button>
            </div>
        `;
        
        return card;
    }

    filterSkills(query) {
        const cards = document.querySelectorAll('.skill-card');
        const lowerQuery = query.toLowerCase();
        
        cards.forEach(card => {
            const title = card.querySelector('.skill-card-title').textContent.toLowerCase();
            const desc = card.querySelector('.skill-card-description').textContent.toLowerCase();
            
            if (title.includes(lowerQuery) || desc.includes(lowerQuery)) {
                card.style.display = '';
            } else {
                card.style.display = 'none';
            }
        });
    }

    toggleSkill(name, enabled) {
        const action = enabled ? 'enable' : 'disable';
        // В реальной реализации API
        this.showToast(`Навык ${name} ${enabled ? 'включён' : 'отключен'}`, 'success');
        this.loadSkills();
    }

    uninstallSkill(name) {
        if (confirm(`Удалить навык ${name}?`)) {
            // В реальной реализации API
            this.showToast(`Навык ${name} удалён`, 'success');
            this.loadSkills();
        }
    }

    // Config
    async loadConfig() {
        try {
            const response = await fetch('/api/v1/config');
            const data = await response.json();
            this.config = data;
            
            const editor = document.getElementById('config-json');
            editor.value = JSON.stringify(data, null, 2);
        } catch (error) {
            console.error('Failed to load config:', error);
        }
    }

    saveConfig() {
        try {
            const editor = document.getElementById('config-json');
            const newConfig = JSON.parse(editor.value);
            
            // В реальной реализации API для сохранения
            this.config = newConfig;
            this.showToast('Конфигурация сохранена', 'success');
        } catch (error) {
            this.showToast('Ошибка валидации JSON: ' + error.message, 'error');
        }
    }

    // Logs
    startLogsStream() {
        const logsStream = document.getElementById('logs-stream');
        logsStream.innerHTML = '';
        
        // Подключаемся к WebSocket для логов
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/logs`;
        
        this.logsWs = new WebSocket(wsUrl);
        
        this.logsWs.onopen = () => {
            this.addLogEntry('info', 'Поток логов запущен (real-time)');
        };
        
        this.logsWs.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            if (msg.type === 'log') {
                this.addLogEntry(msg.payload.level, msg.payload.message);
            }
        };
        
        this.logsWs.onclose = () => {
            this.addLogEntry('warn', 'Поток логов отключён. Переподключение...');
            setTimeout(() => this.startLogsStream(), 3000);
        };
        
        this.logsWs.onerror = (error) => {
            console.error('Logs WebSocket error:', error);
        };
    }

    addLogEntry(level, message) {
        const logsStream = document.getElementById('logs-stream');
        const entry = document.createElement('div');
        entry.className = `log-entry ${level}`;
        
        const timestamp = new Date().toLocaleTimeString();
        entry.innerHTML = `
            <span class="log-timestamp">[${timestamp}]</span>
            <span class="log-level">${level.toUpperCase()}</span>
            <span class="log-message">${message}</span>
        `;
        
        logsStream.appendChild(entry);
        logsStream.scrollTop = logsStream.scrollHeight;
    }

    // Toast notifications
    showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        
        container.appendChild(toast);
        
        setTimeout(() => {
            toast.remove();
        }, 3000);
    }
}

// Modal functions
function openModal() {
    document.getElementById('session-modal').classList.remove('hide');
}

function closeModal() {
    document.getElementById('session-modal').classList.add('hide');
}

// Initialize app
const app = new QwenClawApp();

// Close modal on outside click
window.onclick = function(event) {
    const modal = document.getElementById('session-modal');
    if (event.target === modal) {
        closeModal();
    }
}
