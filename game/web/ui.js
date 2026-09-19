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