export const screens = {
  auth: document.getElementById('auth-screen'),
  register: document.getElementById('register-screen'),
  lobby: document.getElementById('lobby-screen'),
  game: document.getElementById('game-screen')
};

// Управление экранами
export function showScreen(screenName) {
  // Скрываем все экраны
  Object.values(screens).forEach(screen => screen.classList.add('hidden'));
  // Показываем нужный
  if (screens[screenName]) {
    screens[screenName].classList.remove('hidden');
  }
}