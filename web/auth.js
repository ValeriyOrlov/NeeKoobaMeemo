import { showScreen } from "./ui.js";
import { loadProfile, authServer } from "./api.js";

const loginForm = document.getElementById('login-form');
const registerForm = document.getElementById('register-form');

// Переключение между входом и регистрацией
document.getElementById('show-register').addEventListener('click', (e) => {
  e.preventDefault();
  showScreen('register');
});

document.getElementById('show-login').addEventListener('click', (e) => {
  e.preventDefault();
  showScreen('auth');
});

// Регистрация
registerForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const email = document.getElementById('reg-email').value;
  const username = document.getElementById('reg-username').value;
  const password = document.getElementById('reg-password').value;

  try {
    const response = await fetch(`${authServer}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, username, password })
    });

    if (response.ok) {
      alert('На указанную Вами почту улетело письмо!\nЧтобы завершить регистрацию, необходимо пройти по ссылке в письме.');
      showScreen('auth');
    } else {
      const err = await response.text();
      alert('Ошибка регистрации: ' + err);
    }
  } catch (error) {
    console.error('Ошибка сети: ' + error);
  }
});

// Вход
loginForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const email = document.getElementById('login-email').value;
  const password = document.getElementById('login-password').value;

  try {
    const response = await fetch(`${authServer}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });

    if (response.ok) {
      const data = await response.json();
      // Сохраняем JWT токен в память в браузера
      console.log("Ответ Auth-сервера:", data);
      localStorage.setItem('game_token', data.access_token);
      loadProfile(); // Загружаем данные игрока
    } else {
      alert('Неверные данные для входа!');
    }
  } catch (error) {
    console.error('Ошибка сети:', error);
  }
});