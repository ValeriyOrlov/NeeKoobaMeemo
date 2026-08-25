import { showScreen } from "./ui.js";

// Получаем адреса серверов
const gameServer = window.ENV.GAME_SERVER_URL;
export const authServer = window.ENV.AUTH_SERVER_URL;

// Находим элементы на странице
const profileUsername = document.getElementById('profile-username');
const profileBalance = document.getElementById('profile-balance');
const profileStats = document.getElementById('profile-stats');

const verifyEmail = (token) =>
  authServer.get(`/verify?token=${token}`).then(res => res.data)

// Загрузка профиля из игрового сервера
export async function loadProfile() {
  const token = localStorage.getItem('game_token');
  if (!token) {
    showScreen('auth');
    return;
  }

  try {
    // Стучимся на игровой сервер за статистикой
    const response = await fetch(`${gameServer}/api/profile`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    if (response.ok) {
      const profile = await response.json();

      // Заполняем карточку таверны
      profileUsername.innerText = profile.username;
      profileBalance.innerText = profile.balance;
      profileStats.innerText = `${profile.wins} / ${profile.games}`;

      showScreen('lobby');
    } else {
      // Если токен закончился или невалиден
      localStorage.removeItem('game_token');
      showScreen('auth');
    }
  } catch (error) {
    console.error('Ошибка сети:', error);
    showScreen('auth');
  }
}

export async function getLeaderboard() {
  const token = localStorage.getItem('game_token');
  try {
    const response = await fetch(`${gameServer}/api/leaderboard`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    if (response.ok) {
      return await response.json();
    } else {
      console.error('Ошибка получения рейтинга');
      return [];
    }
  } catch (error) {
    console.error('Ошибка сети');
    return [];
  }
}

// Получить список доступных комнат
export async function getRooms() {
  const token = localStorage.getItem('game_token');
  try {
    const response = await fetch(`${gameServer}/api/rooms`, {
      headers: { 'Authorization': `Bearer ${token}`}
    });
    if (response.ok) return await response.json();
    return [];
  } catch (error) {
    console.error('Ошибка получения комнат: ', error);
    return [];
  }
}

// Создать новую комнату со ставкой
export async function createRoomReq(betAmount) {
  const token = localStorage.getItem('game_token');
  const response = await fetch(`${gameServer}/api/rooms`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({ bet_amount: Number(betAmount) })
  });

  if (!response.ok) throw new Error(await response.text());
  return await response.json(); // Ожидаем { room_id: "..." }
}

// Присоединиться к существующей комнате
export async function joinRoomReq(roomId) {
  const token = localStorage.getItem('game_token');
  const response = await fetch(`${gameServer}/api/rooms/join?id=${roomId}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    }
  });

  if (!response.ok) throw new Error(await response.text());
  return await response.json();
}