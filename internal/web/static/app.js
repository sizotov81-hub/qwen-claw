// Qwen-Claw Web UI - JavaScript с аутентификацией

// Состояние
const state = {
    ws: null,
    currentTab: 'chat',
    messages: [],
    token: null,
    secret: null
};

// DOM элементы
const elements = {
    authScreen: document.getElementById('auth-screen'),
    mainApp: document.getElementById('main-app'),
    authForm: document.getElementById('auth-form'),
    secretInput: document.getElementById('secret-input'),
    authError: document.getElementById('auth-error'),
    logoutBtn: document.getElementById('logout-btn'),
    wsStatus: document.getElementById('ws-status'),
    wsText: document.getElementById('ws-text'),
    chatMessages: document.getElementById('chat-messages'),
    chatInput: document.getElementById('chat-input'),
    sendBtn: document.getElementById('send-btn'),
    navBtns: document.querySelectorAll('.nav-btn'),
    tabs: document.querySelectorAll('.tab'),
    modal: document.getElementById('modal'),
    modalTitle: document.getElementById('modal-title'),
    modalForm: document.getElementById('modal-form'),
    modalClose: document.querySelector('.close')
};

// Проверка сохранённой аутентификации
document.addEventListener('DOMContentLoaded', () => {
    const savedToken = localStorage.getItem('qwen_token');
    const savedSecret = localStorage.getItem('qwen_secret');
    
    if (savedToken && savedSecret) {
        state.token = savedToken;
        state.secret = savedSecret;
        showMainApp();
        initWebSocket();
    } else {
        showAuthScreen();
    }
});

// Аутентификация
elements.authForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const secret = elements.secretInput.value.trim();
    if (!secret) return;
    
    try {
        const response = await fetch('/api/auth', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ secret })
        });
        
        const data = await response.json();
        
        if (response.status === 404 || !data.success) {
            showAuthError('Invalid secret phrase');
            return;
        }
        
        // Сохраняем токен и секрет
        state.token = data.token;
        state.secret = secret;
        localStorage.setItem('qwen_token', data.token);
        localStorage.setItem('qwen_secret', secret);
        
        showMainApp();
        initWebSocket();
        loadStatus();
    } catch (err) {
        showAuthError('Connection failed: ' + err.message);
    }
});

// Выход
elements.logoutBtn.addEventListener('click', () => {
    localStorage.removeItem('qwen_token');
    localStorage.removeItem('qwen_secret');
    state.token = null;
    state.secret = null;
    if (state.ws) state.ws.close();
    showAuthScreen();
});

function showAuthScreen() {
    elements.authScreen.style.display = 'flex';
    elements.mainApp.classList.remove('visible');
}

function showMainApp() {
    elements.authScreen.style.display = 'none';
    elements.mainApp.classList.add('visible');
    initNavigation();
    initChat();
    initMemory();
    initTasks();
    loadSkills();
}

function showAuthError(message) {
    elements.authError.textContent = message;
    elements.authError.classList.add('visible');
    setTimeout(() => {
        elements.authError.classList.remove('visible');
    }, 5000);
}

// Навигация
function initNavigation() {
    elements.navBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            switchTab(tab);
        });
    });
}

function switchTab(tab) {
    state.currentTab = tab;
    
    elements.navBtns.forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === tab);
    });
    
    elements.tabs.forEach(t => {
        t.classList.toggle('active', t.id === tab);
    });
    
    // Загружаем данные для вкладки
    if (tab === 'memory') loadMemory();
    if (tab === 'tasks') loadTasks();
    if (tab === 'status') loadStatus();
}

// WebSocket с аутентификацией
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(state.token)}`;
    
    state.ws = new WebSocket(wsUrl);
    
    state.ws.onopen = () => {
        elements.wsStatus.classList.add('connected');
        elements.wsText.textContent = 'Подключено';
    };
    
    state.ws.onclose = () => {
        elements.wsStatus.classList.remove('connected');
        elements.wsText.textContent = 'Отключено';
        // Переподключение через 5 секунд
        setTimeout(initWebSocket, 5000);
    };
    
    state.ws.onerror = () => {
        elements.wsText.textContent = 'Ошибка';
    };
    
    state.ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        handleWSMessage(data);
    };
}

function handleWSMessage(data) {
    if (data.type === 'thinking') {
        // Показываем индикатор "думает"
        showThinkingIndicator();
    } else if (data.type === 'chat_response') {
        // Скрываем индикатор
        hideThinkingIndicator();
        
        // Потоковый вывод с эффектом печати
        if (data.streaming) {
            typeWriterEffect(data.message, 'assistant');
        } else {
            addMessage(data.message, 'user');
            addMessage(data.response || data.message, 'assistant');
        }
        
        // Проверяем confirmation в отдельном сообщении
    } else if (data.type === 'confirmation') {
        // Показываем кнопки подтверждения
        showConfirmationButtons(data.actions);
    } else if (data.type === 'error') {
        hideThinkingIndicator();
        addMessage(data.message, 'error');
    }
}

function showThinkingIndicator() {
    const div = document.createElement('div');
    div.className = 'message assistant thinking';
    div.id = 'thinking-indicator';
    div.innerHTML = '🤔 Думаю...<span class="cursor">▌</span>';
    elements.chatMessages.appendChild(div);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

function hideThinkingIndicator() {
    const indicator = document.getElementById('thinking-indicator');
    if (indicator) {
        indicator.remove();
    }
}

function typeWriterEffect(text, type) {
    const div = document.createElement('div');
    div.className = `message ${type}`;
    elements.chatMessages.appendChild(div);
    
    // Рендерим markdown
    div.innerHTML = marked.parse(text);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

function showConfirmationButtons(actions) {
    const div = document.createElement('div');
    div.className = 'message confirmation';
    
    let html = '<div class="confirmation-box"><strong>⚠️ Требуется подтверждение:</strong><br><br>';
    actions.forEach(action => {
        html += `<div class="confirmation-item">
            <code>${action.id}</code>: ${action.query}
            <button onclick="confirmAction('${action.id}', true)" class="confirm-btn">✅ Подтвердить</button>
            <button onclick="confirmAction('${action.id}', false)" class="reject-btn">❌ Отклонить</button>
        </div>`;
    });
    html += '</div>';
    
    div.innerHTML = html;
    elements.chatMessages.appendChild(div);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

function confirmAction(actionId, confirm) {
    fetch('/api/confirm', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${state.token}`
        },
        body: JSON.stringify({
            action_id: actionId,
            confirm: confirm
        })
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            addMessage(data.data.message, 'assistant');
            // Удаляем кнопки подтверждения
            const confirmationBoxes = document.querySelectorAll('.confirmation-box');
            confirmationBoxes.forEach(box => box.remove());
        } else {
            addMessage(data.error, 'error');
        }
    })
    .catch(err => addMessage(err.message, 'error'));
}

// Чат
function initChat() {
    elements.sendBtn.addEventListener('click', sendMessage);
    elements.chatInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') sendMessage();
    });
}

function sendMessage() {
    const message = elements.chatInput.value.trim();
    if (!message) return;
    
    addMessage(message, 'user');
    elements.chatInput.value = '';
    
    // Отправляем через WebSocket
    if (state.ws && state.ws.readyState === WebSocket.OPEN) {
        state.ws.send(JSON.stringify({
            action: 'chat',
            message: message
        }));
    } else {
        // Fallback через HTTP
        fetch('/api/chat', {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${state.token}`
            },
            body: JSON.stringify({ message })
        })
        .then(res => res.json())
        .then(data => {
            if (data.success) {
                addMessage(data.data.response, 'assistant');
            } else {
                addMessage(data.error, 'error');
            }
        })
        .catch(err => addMessage(err.message, 'error'));
    }
}

function addMessage(text, type) {
    const div = document.createElement('div');
    div.className = `message ${type}`;
    div.textContent = text;
    elements.chatMessages.appendChild(div);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

// Память
function initMemory() {
    document.getElementById('add-memory-btn').addEventListener('click', () => {
        showModal('Добавить в память', [
            { name: 'type', label: 'Тип', type: 'select', options: ['fact', 'context', 'note'] },
            { name: 'content', label: 'Содержимое', type: 'textarea' }
        ], async (data) => {
            const res = await fetch('/api/memory', {
                method: 'POST',
                headers: { 
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${state.token}`
                },
                body: JSON.stringify({ type: data.type, content: data.content })
            });
            const result = await res.json();
            if (result.success) {
                loadMemory();
            } else {
                alert(result.error);
            }
        });
    });
    
    document.getElementById('clear-memory-btn').addEventListener('click', async () => {
        if (!confirm('Вы уверены?')) return;
        await fetch('/api/memory', { 
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${state.token}` }
        });
        loadMemory();
    });
}

async function loadMemory() {
    const res = await fetch('/api/memory', {
        headers: { 'Authorization': `Bearer ${state.token}` }
    });
    const data = await res.json();
    const list = document.getElementById('memory-list');
    
    if (!data.success || !data.data || data.data.length === 0) {
        list.innerHTML = '<p class="empty">Память пуста</p>';
        return;
    }
    
    list.innerHTML = data.data.map(entry => `
        <div class="list-item">
            <h4>[${entry.type}] ${escapeHtml(entry.content)}</h4>
            <div class="meta">
                <span>Создано: ${new Date(entry.created).toLocaleString()}</span>
            </div>
        </div>
    `).join('');
}

// Задачи
function initTasks() {
    document.getElementById('add-task-btn').addEventListener('click', () => {
        showModal('Добавить задачу', [
            { name: 'name', label: 'Имя', type: 'text' },
            { name: 'schedule', label: 'Расписание', type: 'text', placeholder: '@daily или 0 9 * * *' },
            { name: 'command', label: 'Команда', type: 'text' }
        ], async (data) => {
            const res = await fetch('/api/tasks', {
                method: 'POST',
                headers: { 
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${state.token}`
                },
                body: JSON.stringify(data)
            });
            const result = await res.json();
            if (result.success) {
                loadTasks();
            } else {
                alert(result.error);
            }
        });
    });
}

async function loadTasks() {
    const res = await fetch('/api/tasks', {
        headers: { 'Authorization': `Bearer ${state.token}` }
    });
    const data = await res.json();
    const list = document.getElementById('tasks-list');
    
    if (!data.success || !data.data || data.data.length === 0) {
        list.innerHTML = '<p class="empty">Нет задач</p>';
        return;
    }
    
    list.innerHTML = data.data.map(task => `
        <div class="list-item">
            <h4>${task.enabled ? '✅' : '❌'} ${escapeHtml(task.name)}</h4>
            <p><strong>Команда:</strong> ${escapeHtml(task.command)}</p>
            <p><strong>Расписание:</strong> <code>${escapeHtml(task.schedule)}</code></p>
            <div class="meta">
                <span>Запусков: ${task.run_count}</span>
                ${task.next_run ? `<span>След.: ${new Date(task.next_run).toLocaleString()}</span>` : ''}
            </div>
        </div>
    `).join('');
}

// Навыки
async function loadSkills() {
    const res = await fetch('/api/skills', {
        headers: { 'Authorization': `Bearer ${state.token}` }
    });
    const data = await res.json();
    const list = document.getElementById('skills-list');
    
    if (!data.success || !data.data) {
        list.innerHTML = '<p class="empty">Нет навыков</p>';
        return;
    }
    
    list.innerHTML = data.data.map(skill => `
        <div class="list-item">
            <h4>${skill.enabled ? '✅' : '❌'} ${escapeHtml(skill.name)}</h4>
            <p>${escapeHtml(skill.description)}</p>
            <div class="meta">
                <span>Команды: ${skill.commands.join(', ')}</span>
            </div>
        </div>
    `).join('');
}

// Статус
async function loadStatus() {
    const res = await fetch('/api/status', {
        headers: { 'Authorization': `Bearer ${state.token}` }
    });
    const data = await res.json();
    const grid = document.getElementById('status-content');
    
    if (!data.success || !data.data) {
        grid.innerHTML = '<p class="empty">Не удалось загрузить статус</p>';
        return;
    }
    
    const d = data.data;
    grid.innerHTML = `
        <div class="status-card">
            <h3>Qwen CLI</h3>
            <div class="value ${d.qwen_available ? 'success' : 'error'}">
                ${d.qwen_available ? '✓ Доступен' : '✗ Не доступен'}
            </div>
        </div>
        <div class="status-card">
            <h3>Путь к Qwen</h3>
            <div class="value" style="font-size: 1rem;">${escapeHtml(d.qwen_path)}</div>
        </div>
        <div class="status-card">
            <h3>Модель</h3>
            <div class="value" style="font-size: 1rem;">${escapeHtml(d.model || 'по умолчанию')}</div>
        </div>
        <div class="status-card">
            <h3>Записей в памяти</h3>
            <div class="value">${d.memory_entries}</div>
        </div>
        <div class="status-card">
            <h3>Задач</h3>
            <div class="value">${d.tasks_count}</div>
        </div>
        <div class="status-card">
            <h3>Навыков</h3>
            <div class="value">${d.skills_count}</div>
        </div>
        <div class="status-card">
            <h3>WebSocket клиенты</h3>
            <div class="value">${d.ws_clients}</div>
        </div>
    `;
}

// Модальное окно
function showModal(title, fields, onSubmit) {
    elements.modalTitle.textContent = title;
    elements.modalForm.innerHTML = '';
    
    fields.forEach(field => {
        const label = document.createElement('label');
        label.textContent = field.label;
        elements.modalForm.appendChild(label);
        
        let input;
        if (field.type === 'select') {
            input = document.createElement('select');
            field.options.forEach(opt => {
                const option = document.createElement('option');
                option.value = opt;
                option.textContent = opt;
                input.appendChild(option);
            });
        } else if (field.type === 'textarea') {
            input = document.createElement('textarea');
            input.rows = 4;
        } else {
            input = document.createElement('input');
            input.type = field.type || 'text';
        }
        
        input.name = field.name;
        if (field.placeholder) input.placeholder = field.placeholder;
        elements.modalForm.appendChild(input);
    });
    
    const submitBtn = document.createElement('button');
    submitBtn.type = 'submit';
    submitBtn.className = 'btn btn-primary';
    submitBtn.textContent = 'Сохранить';
    elements.modalForm.appendChild(submitBtn);
    
    elements.modalForm.onsubmit = async (e) => {
        e.preventDefault();
        const formData = new FormData(elements.modalForm);
        const data = Object.fromEntries(formData);
        await onSubmit(data);
        elements.modal.style.display = 'none';
    };
    
    elements.modal.style.display = 'block';
}

elements.modalClose.addEventListener('click', () => {
    elements.modal.style.display = 'none';
});

window.addEventListener('click', (e) => {
    if (e.target === elements.modal) {
        elements.modal.style.display = 'none';
    }
});

// Утилиты
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
