// Load тесты для Qwen-Claw микросервисов
// Запуск: k6 run tests/load/load_test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Кастомные метрики
const errorRate = new Rate('errors');
const sessionCreateRate = new Rate('session_created');

// Конфигурация
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const GRPC_ADDR = __ENV.GRPC_ADDR || 'localhost:50051';

// Настройки теста
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Разогрев до 10 RPS
    { duration: '1m', target: 50 },    // Нагрузка до 50 RPS
    { duration: '2m', target: 50 },    // Пиковая нагрузка
    { duration: '1m', target: 100 },   // Стресс тест
    { duration: '30s', target: 0 },    // Остывание
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% запросов < 500ms
    http_req_failed: ['rate<0.1'],     // Ошибки < 10%
    errors: ['rate<0.1'],              // Кастомная метрика ошибок
  },
};

// Генерация случайного user ID
function randomUserID() {
  return `user_${Math.floor(Math.random() * 10000)}`;
}

// Генерация случайного title
function randomTitle() {
  const titles = ['Test Session', 'Integration Test', 'Load Test', 'Chat Session'];
  return titles[Math.floor(Math.random() * titles.length)];
}

// Тестовая группа 1: Health checks
export function healthCheck() {
  const res = http.get(`${BASE_URL}/health`);
  
  const success = check(res, {
    'health status is 200': (r) => r.status === 200,
    'health response time < 100ms': (r) => r.timings.duration < 100,
  });
  
  errorRate.add(!success);
  return success;
}

// Тестовая группа 2: Создание сессии
export function createSession() {
  const payload = JSON.stringify({
    user_id: randomUserID(),
    title: randomTitle(),
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/sessions`, payload, params);
  
  const success = check(res, {
    'create session status is 201': (r) => r.status === 201,
    'session ID present': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id && body.id !== '';
      } catch (e) {
        return false;
      }
    },
  });
  
  sessionCreateRate.add(success);
  errorRate.add(!success);
  
  return res;
}

// Тестовая группа 3: Получение сессии
export function getSession(sessionId) {
  if (!sessionId) {
    return null;
  }
  
  const res = http.get(`${BASE_URL}/api/v1/sessions/${sessionId}`);
  
  const success = check(res, {
    'get session status is 200': (r) => r.status === 200,
  });
  
  errorRate.add(!success);
  return res;
}

// Тестовая группа 4: Отправка сообщения
export function sendMessage(sessionId) {
  if (!sessionId) {
    return null;
  }
  
  const payload = JSON.stringify({
    session_id: sessionId,
    content: 'Hello, this is a load test message!',
    model: 'gpt-4',
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/chat`, payload, params);
  
  const success = check(res, {
    'send message status is 200': (r) => r.status === 200,
    'response time < 1s': (r) => r.timings.duration < 1000,
  });
  
  errorRate.add(!success);
  return res;
}

// Тестовая группа 5: Поиск в памяти
export function searchMemory(userId) {
  const res = http.get(`${BASE_URL}/api/v1/memory?user_id=${userId}&query=test`);
  
  const success = check(res, {
    'search status is 200': (r) => r.status === 200,
  });
  
  errorRate.add(!success);
  return res;
}

// Основной сценарий
export default function () {
  // Health check
  healthCheck();
  sleep(0.5);
  
  // Создание сессии
  const createRes = createSession();
  let sessionId = null;
  
  if (createRes && createRes.status === 201) {
    try {
      const body = JSON.parse(createRes.body);
      sessionId = body.id;
    } catch (e) {
      // Ignore
    }
  }
  
  sleep(0.5);
  
  // Получение сессии
  if (sessionId) {
    getSession(sessionId);
    sleep(0.5);
    
    // Отправка сообщения
    sendMessage(sessionId);
    sleep(0.5);
  }
  
  // Поиск в памяти
  searchMemory(randomUserID());
  sleep(1);
}

// Setup - выполняется один раз перед всеми тестами
export function setup() {
  console.log('Starting load test...');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`gRPC Address: ${GRPC_ADDR}`);
  
  // Проверка доступности сервиса
  const res = http.get(`${BASE_URL}/health`);
  if (res.status !== 200) {
    console.warn('Service may not be available!');
  }
  
  return { startTime: new Date() };
}

// Teardown - выполняется один раз после всех тестов
export function teardown(data) {
  const endTime = new Date();
  const duration = (endTime - data.startTime) / 1000;
  
  console.log(`Load test completed in ${duration}s`);
  console.log(`Error rate: ${errorRate.rate.toFixed(4)}`);
  console.log(`Session create rate: ${sessionCreateRate.rate.toFixed(4)}`);
}
