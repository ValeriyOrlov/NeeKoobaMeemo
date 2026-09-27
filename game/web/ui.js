import { setMusicMode } from "./music.js";

export const screens = {
  auth: document.getElementById('auth-screen'),
  register: document.getElementById('register-screen'),
  lobby: document.getElementById('lobby-screen'),
  game: document.getElementById('game-screen')
};

// Добавьте эти переменные в ui.js
export let isGameActive = false;

export function setIsGameActive(status) {
    isGameActive = status;
}

// Управление экранами
export function showScreen(screenName) {
  // Если пытаются открыть лобби, но игра уже активна - блокируем
  if (screenName === 'lobby' && isGameActive) {
    console.log("Блокировка показа лобби: активна игра");
    return;
  }
  // Скрываем все экраны
  Object.values(screens).forEach(screen => screen.classList.add('hidden'));
  // Показываем нужный
  if (screens[screenName]) {
    screens[screenName].classList.remove('hidden');
  }

  if (screenName === 'lobby') {
      setMusicMode('lobby');
    } else if (screenName === 'game') {
      setMusicMode('game');
  }
}

export function showWaitingOverlay(text = "Ожидание второго игрока...") {
  let overlay = document.getElementById('waiting-overlay');
  
  if (!overlay) {
    overlay = document.createElement('div');
    overlay.id = 'waiting-overlay';
    overlay.className = 'waiting-overlay';
    overlay.innerHTML = `
      <div class="waiting-content">
        <div class="spinner"></div>
        <p id="waiting-overlay-text">${text}</p>
        <button id="cancel-waiting-btn" class="btn btn-secondary">Отменить</button>
      </div>
    `;
    document.body.appendChild(overlay);
  } else {
    document.getElementById('waiting-overlay-text').innerText = text;
    overlay.style.display = 'flex';
  }
}

export function hideWaitingOverlay() {
  const overlay = document.getElementById('waiting-overlay');
  if (overlay) {
    overlay.style.display = 'none';
  }
}

export function showGameStatus(text, duration = 4000) {
  let statusEl = document.getElementById('game-status-toast');

  if (!statusEl) {
    statusEl = document.createElement('div');
    statusEl.id = 'game-status-toast';
    statusEl.className = 'status-toast';
    document.body.appendChild(statusEl);
  }

  statusEl.innerText = text;
  statusEl.classList.add('visible');

  if (statusEl.hideTimeout) {
    clearTimeout(statusEl.hideTimeout);
  }

  statusEl.hideTimeout = setTimeout(() => {
    statusEl.classList.remove('visible');
  }, duration);
}

// ui.js

let reconnectOverlay = null;

export function showReconnectingState(attempt, delayMs, onCancelCallback) {
  // Блокируем интерактивные элементы (кнопки броска, инпуты)
  document.querySelectorAll('.controls-area').forEach(el => {
    el.disabled = true;
  });

  if (!reconnectOverlay) {
    reconnectOverlay = document.createElement('div');
    reconnectOverlay.className = 'offline-overlay';
    // Базовые стили для затемнения и блокировки кликов
    reconnectOverlay.style.cssText = `
      position: absolute; top: 0; left: 0; right: 0; bottom: 0;
      background: rgba(0, 0, 0, 0.7); z-index: 1000;
      display: flex; flex-direction: column; 
      align-items: center; justify-content: center;
      color: white; font-family: sans-serif;
    `;
    document.getElementById('game-screen').appendChild(reconnectOverlay);
  }

  const seconds = Math.round(delayMs / 1000);
  reconnectOverlay.innerHTML = `
    <h3>⚠️ Связь потеряна</h3>
    <p>Попытка переподключения #${attempt} через ${seconds} сек...</p>
    <button id="abort-reconnect-btn" style="margin-top: 15px; padding: 8px 16px; background: #ff4444; color: white; border: none; cursor: pointer;">
      Покинуть игру
    </button>
  `;

  document.getElementById('abort-reconnect-btn').onclick = () => {
    hideReconnectingState();
    if (onCancelCallback) onCancelCallback();
  };
}

export function hideReconnectingState() {
  if (reconnectOverlay) {
    reconnectOverlay.remove();
    reconnectOverlay = null;
  }
  
  // Разблокируем элементы
  document.querySelectorAll('.controls-area').forEach(el => {
    el.disabled = false;
  });
}

export function showOpponentOfflineWarning(onLeaveCallback) {
  // Аналогичный оверлей, но для ожидания противника
  showReconnectingState('Ожидание', 60000, onLeaveCallback);
  reconnectOverlay.innerHTML = `
    <h3>⚠️ Противник отключился</h3>
    <p>Ожидаем возвращения игрока (максимум 60 секунд)...</p>
    <button id="leave-abandoned-game-btn" style="margin-top: 15px; padding: 8px 16px; background: #ff4444; color: white; border: none; cursor: pointer;">
      Выйти в лобби
    </button>
  `;
  document.getElementById('leave-abandoned-game-btn').onclick = () => {
    hideReconnectingState();
    if (onLeaveCallback) onLeaveCallback();
  };
}