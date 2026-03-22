// ============================================
// Qwen-Claw Web UI - Modern App
// ============================================

// Состояние приложения
const state = {
    token: null,
    ws: null,
    isConnecting: false,
    chatHistory: [],
    currentChat: null,
    settings: {
        theme: 'dark',
        typingSpeed: 10,
        showContext: true
    }
};

// DOM элементы
const elements = {
    authScreen: document.getElementById('auth-screen'),
    mainApp: document.getElementById('main-app'),
    authForm: document.getElementById('auth-form'),
    secretInput: document.getElementById('secret-input'),
    authError: document.getElementById('auth-error'),
    chatMessages: document.getElementById('chat-messages'),
    chatInput: document.getElementById('chat-input'),
    sendBtn: document.getElementById('send-btn'),
    newChatBtn: document.getElementById('new-chat-btn'),
    clearChatBtn: document.getElementById('clear-chat-btn'),
    exportChatBtn: document.getElementById('export-chat-btn'),
    chatHistory: document.getElementById('chat-history'),
    memoryCount: document.getElementById('memory-count'),
    wsStatus: document.getElementById('ws-status'),
    wsText: document.getElementById('ws-text'),
    logoutBtn: document.getElementById('logout-btn'),
    settingsBtn: document.getElementById('settings-btn'),
    settingsModal: document.getElementById('settings-modal'),
    closeSettings: document.getElementById('close-settings'),
    themeSelect: document.getElementById('theme-select'),
    typingSpeed: document.getElementById('typing-speed'),
    typingSpeedValue: document.getElementById('typing-speed-value'),
    showContextToggle: document.getElementById('show-context-toggle'),
    suggestionBtns: document.querySelectorAll('.suggestion-btn')
};

// ============================================
// Инициализация
// ============================================
function init() {
    loadSettings();
    loadToken();
    setupEventListeners();
    
    if (state.token) {
        showMainApp();
        connectWebSocket();
    }
}

function loadToken() {
    state.token = localStorage.getItem('qwenclaw_token');
}

function saveToken(token) {
    localStorage.setItem('qwenclaw_token', token);
    state.token = token;
}

function loadSettings() {
    const saved = localStorage.getItem('qwenclaw_settings');
    if (saved) {
        state.settings = { ...state.settings, ...JSON.parse(saved) };
        applySettings();
    }
}

function saveSettings() {
    localStorage.setItem('qwenclaw_settings', JSON.stringify(state.settings));
}

function applySettings() {
    // Тема
    if (state.settings.theme === 'system') {
        const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        document.documentElement.setAttribute('data-theme', systemTheme);
    } else {
        document.documentElement.setAttribute('data-theme', state.settings.theme);
    }
    
    // UI элементы
    elements.themeSelect.value = state.settings.theme;
    elements.typingSpeed.value = state.settings.typingSpeed;
    elements.typingSpeedValue.textContent = `${state.settings.typingSpeed} мс/символ`;
    elements.showContextToggle.checked = state.settings.showContext;
}

// ============================================
// Event Listeners
// ============================================
function setupEventListeners() {
    elements.authForm.addEventListener('submit', handleAuth);
    elements.chatInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });
    elements.sendBtn.addEventListener('click', sendMessage);
    elements.newChatBtn.addEventListener('click', startNewChat);
    elements.clearChatBtn.addEventListener('click', clearChat);
    elements.exportChatBtn.addEventListener('click', exportChat);
    elements.logoutBtn.addEventListener('click', logout);
    elements.settingsBtn.addEventListener('click', () => {
        elements.settingsModal.classList.remove('hidden');
    });
    elements.closeSettings.addEventListener('click', () => {
        elements.settingsModal.classList.add('hidden');
    });
    elements.themeSelect.addEventListener('change', (e) => {
        state.settings.theme = e.target.value;
        saveSettings();
        applySettings();
    });
    elements.typingSpeed.addEventListener('input', (e) => {
        state.settings.typingSpeed = parseInt(e.target.value);
        elements.typingSpeedValue.textContent = `${state.settings.typingSpeed} мс/символ`;
        saveSettings();
    });
    elements.showContextToggle.addEventListener('change', (e) => {
        state.settings.showContext = e.target.checked;
        saveSettings();
    });
    
    // Кнопки предложений
    elements.suggestionBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const query = btn.dataset.query;
            elements.chatInput.value = query;
            sendMessage();
        });
    });
    
    // Авто-ресайз textarea
    elements.chatInput.addEventListener('input', function() {
        this.style.height = 'auto';
        this.style.height = Math.min(this.scrollHeight, 200) + 'px';
    });
}

// ============================================
// Аутентификация
// ============================================
function handleAuth(e) {
    e.preventDefault();
    
    const secret = elements.secretInput.value.trim();
    if (!secret) {
        showAuthError('Введите секретную фразу');
        return;
    }
    
    // Сохраняем токен
    saveToken(secret);
    
    // Показываем основной интерфейс
    showMainApp();
    
    // Подключаем WebSocket
    connectWebSocket();
}

function showAuthError(message) {
    elements.authError.textContent = message;
    elements.authError.style.display = 'block';
    setTimeout(() => {
        elements.authError.style.display = 'none';
    }, 5000);
}

function showMainApp() {
    elements.authScreen.classList.add('hidden');
    elements.mainApp.classList.remove('hidden');
}

function logout() {
    localStorage.removeItem('qwenclaw_token');
    state.token = null;
    
    if (state.ws) {
        state.ws.close();
    }
    
    elements.mainApp.classList.add('hidden');
    elements.authScreen.classList.remove('hidden');
    elements.secretInput.value = '';
}

// ============================================
// WebSocket
// ============================================
function connectWebSocket() {
    if (state.isConnecting) return;
    
    state.isConnecting = true;
    updateWSStatus('connecting');
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    state.ws = new WebSocket(wsUrl);
    
    state.ws.onopen = () => {
        state.isConnecting = false;
        updateWSStatus('connected');
        loadChatHistory();
    };
    
    state.ws.onmessage = (event) => {
        handleWSMessage(JSON.parse(event.data));
    };
    
    state.ws.onclose = () => {
        state.isConnecting = false;
        updateWSStatus('disconnected');
        
        // Переподключение через 5 секунд
        setTimeout(connectWebSocket, 5000);
    };
    
    state.ws.onerror = () => {
        updateWSStatus('error');
    };
}

function updateWSStatus(status) {
    elements.wsStatus.className = 'status-dot';
    
    switch (status) {
        case 'connecting':
            elements.wsStatus.classList.add('connected');
            elements.wsText.textContent = 'Подключение...';
            break;
        case 'connected':
            elements.wsStatus.classList.add('connected');
            elements.wsText.textContent = 'Подключено';
            break;
        case 'disconnected':
            elements.wsStatus.classList.add('disconnected');
            elements.wsText.textContent = 'Отключено';
            break;
        case 'error':
            elements.wsStatus.classList.add('disconnected');
            elements.wsText.textContent = 'Ошибка';
            break;
    }
}

function handleWSMessage(data) {
    console.log('WS Message:', data);
    
    switch (data.type) {
        case 'thinking':
            showTypingIndicator();
            break;
            
        case 'token':
            appendToken(data.token);
            break;
            
        case 'chat_response':
            hideTypingIndicator();
            if (data.streaming) {
                // Streaming уже обработан через token events
            } else {
                addMessage(data.response, 'assistant');
            }
            break;
            
        case 'error':
            hideTypingIndicator();
            addMessage(`❌ Ошибка: ${data.message}`, 'error');
            break;
            
        case 'confirmation':
            showConfirmation(data);
            break;
    }
}

// ============================================
// Отправка сообщений
// ============================================
function sendMessage() {
    const message = elements.chatInput.value.trim();
    if (!message) return;
    
    // Добавляем сообщение пользователя
    addMessage(message, 'user');
    elements.chatInput.value = '';
    elements.chatInput.style.height = 'auto';
    
    // Отправляем через WebSocket
    if (state.ws && state.ws.readyState === WebSocket.OPEN) {
        state.ws.send(JSON.stringify({
            action: 'chat',
            message: message
        }));
    } else {
        // Fallback через HTTP
        sendViaHTTP(message);
    }
}

async function sendViaHTTP(message) {
    try {
        const response = await fetch('/api/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${state.token}`
            },
            body: JSON.stringify({ message })
        });
        
        const data = await response.json();
        if (data.success) {
            addMessage(data.data.response, 'assistant');
        } else {
            addMessage(`❌ Ошибка: ${data.error}`, 'error');
        }
    } catch (err) {
        addMessage(`❌ Ошибка: ${err.message}`, 'error');
    }
}

// ============================================
// Отображение сообщений
// ============================================
function addMessage(text, type) {
    // Скрываем приветствие если есть
    const welcome = elements.chatMessages.querySelector('.welcome-message');
    if (welcome) {
        welcome.style.display = 'none';
    }
    
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${type}`;
    
    const avatar = type === 'user' ? '👤' : '🤖';
    
    messageDiv.innerHTML = `
        <div class="message-avatar">${avatar}</div>
        <div class="message-content">${marked.parse(text)}</div>
    `;
    
    elements.chatMessages.appendChild(messageDiv);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
    
    // Подсветка кода
    messageDiv.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
    });
}

let currentMessageDiv = null;
let currentMessageText = '';

function showTypingIndicator() {
    const indicator = document.createElement('div');
    indicator.id = 'typing-indicator';
    indicator.className = 'message assistant';
    indicator.innerHTML = `
        <div class="message-avatar">🤖</div>
        <div class="message-content">
            <div class="typing-indicator">
                <div class="typing-dot"></div>
                <div class="typing-dot"></div>
                <div class="typing-dot"></div>
            </div>
        </div>
    `;
    
    elements.chatMessages.appendChild(indicator);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

function hideTypingIndicator() {
    const indicator = document.getElementById('typing-indicator');
    if (indicator) {
        indicator.remove();
    }
}

function appendToken(token) {
    if (!currentMessageDiv) {
        currentMessageDiv = document.createElement('div');
        currentMessageDiv.className = 'message assistant';
        currentMessageDiv.innerHTML = `
            <div class="message-avatar">🤖</div>
            <div class="message-content"></div>
        `;
        elements.chatMessages.appendChild(currentMessageDiv);
    }
    
    currentMessageText += token;
    const contentDiv = currentMessageDiv.querySelector('.message-content');
    contentDiv.innerHTML = marked.parse(currentMessageText);
    
    // Подсветка кода
    contentDiv.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
    });
    
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

// ============================================
// Управление чатом
// ============================================
function startNewChat() {
    elements.chatMessages.innerHTML = `
        <div class="welcome-message">
            <div class="welcome-icon">🤖</div>
            <h2>Добро пожаловать в Qwen-Claw!</h2>
            <p>Я ваш AI-помощник с персистентной памятью.</p>
            <div class="welcome-suggestions">
                <button class="suggestion-btn" data-query="Расскажи о своих возможностях">
                    💡 Расскажи о своих возможностях
                </button>
                <button class="suggestion-btn" data-query="Запомни: я изучаю Go">
                    🧠 Запомни факт
                </button>
                <button class="suggestion-btn" data-query="Выполни команду: pwd">
                    ⚡ Выполнить команду
                </button>
                <button class="suggestion-btn" data-query="Какие задачи запланированы?">
                    📋 Показать задачи
                </button>
            </div>
        </div>
    `;
    
    currentMessageDiv = null;
    currentMessageText = '';
    
    // Пересоздаём обработчики для кнопок
    document.querySelectorAll('.suggestion-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const query = btn.dataset.query;
            elements.chatInput.value = query;
            sendMessage();
        });
    });
}

function clearChat() {
    if (confirm('Вы уверены что хотите очистить чат?')) {
        startNewChat();
    }
}

function exportChat() {
    const messages = [];
    elements.chatMessages.querySelectorAll('.message').forEach(msg => {
        const type = msg.classList.contains('user') ? 'user' : 'assistant';
        const text = msg.querySelector('.message-content').textContent;
        messages.push({ type, text });
    });
    
    const blob = new Blob([JSON.stringify(messages, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `chat-${new Date().toISOString().split('T')[0]}.json`;
    a.click();
    URL.revokeObjectURL(url);
}

async function loadChatHistory() {
    try {
        const response = await fetch('/api/memory', {
            headers: {
                'Authorization': `Bearer ${state.token}`
            }
        });
        
        const data = await response.json();
        if (data.success) {
            state.memoryCount = data.data.length;
            elements.memoryCount.textContent = data.data.length;
        }
    } catch (err) {
        console.error('Failed to load chat history:', err);
    }
}

// ============================================
// Подтверждения
// ============================================
function showConfirmation(data) {
    const confirmDiv = document.createElement('div');
    confirmDiv.className = 'message confirmation';
    confirmDiv.innerHTML = `
        <div class="message-avatar">⚠️</div>
        <div class="message-content">
            <p><strong>Требуется подтверждение действия</strong></p>
            <p>Найдено ожид. действий: ${data.actions}</p>
            <button class="confirm-action-btn" onclick="viewPendingActions()">
                👁️ Просмотреть действия
            </button>
        </div>
    `;
    
    elements.chatMessages.appendChild(confirmDiv);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

async function viewPendingActions() {
    try {
        const response = await fetch('/api/confirmations', {
            headers: {
                'Authorization': `Bearer ${state.token}`
            }
        });
        
        const data = await response.json();
        if (data.success && data.data.length > 0) {
            const actions = data.data;
            const html = actions.map(action => `
                <div class="confirmation-item">
                    <code>${action.id}</code>
                    <p>${action.query}</p>
                    <div class="confirmation-buttons">
                        <button onclick="confirmAction('${action.id}', true)" class="confirm-btn">✅ Подтвердить</button>
                        <button onclick="confirmAction('${action.id}', false)" class="reject-btn">❌ Отклонить</button>
                    </div>
                </div>
            `).join('');
            
            addMessage(`
                <div class="pending-actions">
                    <h4>Ожидающие действия:</h4>
                    ${html}
                </div>
            `, 'assistant');
        }
    } catch (err) {
        addMessage(`❌ Ошибка: ${err.message}`, 'error');
    }
}

async function confirmAction(actionId, confirm) {
    try {
        const response = await fetch('/api/confirm', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${state.token}`
            },
            body: JSON.stringify({
                action_id: actionId,
                confirm: confirm
            })
        });
        
        const data = await response.json();
        if (data.success) {
            addMessage(`✅ Действие ${confirm ? 'подтверждено' : 'отклонено'}`, 'assistant');
        } else {
            addMessage(`❌ Ошибка: ${data.error}`, 'error');
        }
    } catch (err) {
        addMessage(`❌ Ошибка: ${err.message}`, 'error');
    }
}

// ============================================
// Запуск
// ============================================
document.addEventListener('DOMContentLoaded', init);
