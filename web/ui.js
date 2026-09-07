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