import { showScreen } from "./ui.js";

// Получаем адреса серверов
const gameServer = window.ENV.GAME_SERVER_URL;
export const authServer = window.ENV.AUTH_SERVER_URL;

// Находим элементы на странице
const profileUsername = document.getElementById('profile-username');
const profileBalance = document.getElementById('profile-balance');
const profileStats = document.getElementById('profile-stats');
const lobbyAvatar = document.getElementById('lobby-avatar-img');

// Предотвращение параллельных запросов на обновление токена
let isRefreshing = false;
let refreshPromise = null;

// Загрузка профиля из игрового сервера
export async function loadProfile() {
  const token = localStorage.getItem('game_token');
  if (!token) {
    showScreen('auth');
    return;
  }

  try {
    // Стучимся на игровой сервер за статистикой
    const response = await fetchWithAuth(`${gameServer}/api/profile`);
    if (response.ok) {
      const profile = await response.json();

      // Обновляем UI баланса
      document.getElementById('profile-balance').innerText = profile.balance;

      // Получаем сохраненную дату последнего получения бонуса
      const savedClaimDate = localStorage.getItem('last_weekly_claim');

      // Сравниваем: если дата изменилась (и это не самый первый вход), показываем алерт
      if (savedClaimDate && savedClaimDate !== profile.last_weekly_claim) {
          alert("Еженедельное подкрепление! Вам зачислено золото от короны.");
      }
      
      // Обновляем сохраненную дату
      localStorage.setItem('last_weekly_claim', profile.last_weekly_claim);
      // Заполняем карточку таверны
      profileUsername.innerText = profile.username;
      profileBalance.innerText = profile.balance;
      profileStats.innerText = `${profile.wins} / ${profile.games}`;
      lobbyAvatar.src = `../pictures/avatars/${profile.avatar}.jpg`;

      showScreen('lobby');
    } else {
      // Если токен закончился или невалиден
      localStorage.removeItem('game_token');
      localStorage.removeItem('refresh_token');
      showScreen('auth');
    }
  } catch (error) {
    console.error('Ошибка сети:', error);
    showScreen('auth');
  }
}

export async function getLeaderboard() {
  try {
    const response = await fetchWithAuth(`${gameServer}/api/leaderboard`);
    if (response.ok) {
      return await response.json();
    } else {
      console.error('Ошибка получения рейтинга');
      return [];
    }
  } catch (error) {
    console.error('Ошибка сети:', error);
    return [];
  }
}

// Получить список доступных комнат
export async function getRooms() {
  try {
    const response = await fetchWithAuth(`${gameServer}/api/rooms`);
    if (response.ok) return await response.json();
    return [];
  } catch (error) {
    console.error('Ошибка получения комнат: ', error);
    return [];
  }
}

// Создать новую комнату со ставкой
export async function createRoomReq(betAmount, targetScore) {
  const response = await fetchWithAuth(`${gameServer}/api/rooms`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ bet_amount: Number(betAmount), target_score: Number(targetScore) })
  });

  if (!response.ok) throw new Error(await response.text());
  return await response.json(); // Ожидаем { room_id: "..." }
}

// Присоединиться к существующей комнате
export async function joinRoomReq(roomId) {
  const response = await fetchWithAuth(`${gameServer}/api/rooms/join?id=${roomId}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    }
  });

  if (!response.ok) throw new Error(await response.text());
  return await response.json();
}

export async function refreshToken() {
  // Если процесс обновления уже запущен, возвращаем существующий Promise
  if (isRefreshing) {
    return refreshPromise;
  }

  isRefreshing = true;

  refreshPromise = (async () => {
    const refresh = localStorage.getItem('refresh_token');
    if (!refresh) {
      isRefreshing = false;
      refreshPromise = null;
      return null;
    }

    try {
      const response = await fetch(`${authServer}/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refresh })
      });

      if (response.ok) {
        const data = await response.json();
        localStorage.setItem('game_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        return data.access_token;
      }
    } catch (error) {
      console.error('Ошибка ротации токенов:', error);
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }

    // Если обновление не удалось — чистим хранилище
    localStorage.removeItem('game_token');
    localStorage.removeItem('refresh_token');
    showScreen('auth');
    return null;
  })();

  return refreshPromise;
}

export async function fetchWithAuth(url, options = {}) {
  let token = localStorage.getItem('game_token');

  options.headers = {
    ...options.headers,
    'Authorization': `Bearer ${token}`
  };

  let response = await fetch(url, options);

  // если сервер вернул 401 (токен протух), пытаемся обновить
  if (response.status === 401) {
    const newToken = await refreshToken();
    if (newToken) {
      options.headers['Authorization'] = `Bearer ${newToken}`;
      response = await fetch(url, options); // повторяем исхоный запрос
    }
  }

  return response;
}

export async function getValidToken() {
  let token = localStorage.getItem('game_token');
  if (!token) return null;

  try {
    // Декодируем JWT (2-я часть токена в base64) без обращения к серверу
    const payload = JSON.parse(atob(token.split('.')[1]));
    const isExpired = (payload.exp * 1000) < Date.now();

    if (isExpired) {
      return await refreshToken(); // Если истёк - обновляем до открытия сокета
    }
    return token;
  } catch (e) {
    return await refreshToken();
  }
}
