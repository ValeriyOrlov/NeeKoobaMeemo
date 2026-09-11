import { showScreen } from "./ui.js";
import {
  getLeaderboard,
  getRooms, 
  createRoomReq,
  joinRoomReq,
  getValidToken,
} from "./api.js";
import { connectWebSocket } from "./ws.js";
import { toggleMusic } from "./music.js";

(async () => {
  // getValidToken проверяет срок годности и при необходимости делает ротацию токенов
  const token = await getValidToken();

  if (token) {
    console.log("Загрузка страницы: ищем активную игру...");
    connectWebSocket(); // запускаем без roomId
  }
})

const btnCreateGame = document.getElementById('btn-create-game');
const btnFindGames = document.getElementById('btn-find-games');
const btnLeaderboard = document.getElementById('btn-leaderboard');
const btnOpenRules = document.getElementById('btn-rules');
const logoutBtn = document.getElementById('btn-logout');

const leaderboardModal = document.getElementById('leaderboard-modal');
const leaderboardBody = document.getElementById('leaderboard-body');
const btnCloseLeaderboard = document.getElementById('btn-close-leaderboard');

const createRoomModal = document.getElementById('create-room-modal');
const findRoomsModal = document.getElementById('find-rooms-modal');
const rulesModal = document.getElementById('rules-modal');
const roomsListDOM = document.getElementById('rooms-list');
const roomBetInput = document.getElementById('room-bet-input');
const btnConfirmCreate = document.getElementById('btn-confirm-create');
const btnCloseCreate = document.getElementById('btn-close-createModal');
const btnCloseFind = document.getElementById('btn-close-find');

const avatarPickerModal = document.getElementById('avatar-picker-modal');
const lobbyAvatarBtn = document.getElementById('lobby-avatar-btn');
const closeAvatarBtn = document.getElementById('btn-close-avatar-picker');
const closeRulesBtn = document.getElementById('btn-close-rules');

const musicBtn = document.getElementById('music-toggle-btn');

musicBtn.addEventListener('click', () => {
    const isPlaying = toggleMusic();
    // Визуальная обратная связь
    if (isPlaying) {
        musicBtn.innerText = "🪕 Музыка (Вкл)";
        musicBtn.style.borderColor = "#2ecc71"; 
    } else {
        musicBtn.innerText = "🪕 Музыка (Выкл)";
        musicBtn.style.borderColor = "";
    }
});

// Открытие окна при клике на аватар в лобби
lobbyAvatarBtn.addEventListener('click', () => {
    avatarPickerModal.showModal();
});

closeAvatarBtn.addEventListener('click', () => {
    avatarPickerModal.close();
});

// Выбор аватара
document.querySelectorAll('.avatar-option').forEach(img => {
    img.addEventListener('click', async (e) => {
        const selectedAvatar = e.target.dataset.avatar;
        const token = localStorage.getItem('game_token');

        // 1. Сохраняем на бэкенд
        const response = await fetch('/api/user/avatar', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ avatar: selectedAvatar })
        });

        if (response.ok) {
            // 2. Обновляем картинку в лобби
            document.getElementById('lobby-avatar-img').src = `../pictures/avatars/${selectedAvatar}.jpg`;
            avatarPickerModal.close();
        }
    });
});

btnOpenRules.addEventListener('click', () => rulesModal.showModal());
closeRulesBtn.addEventListener('click', () => rulesModal.close());

export const logout = () => {
  localStorage.removeItem('game_token');
  showScreen('auth');
};

export const leaderbord = async () => {
  const users = await getLeaderboard();
  leaderboardBody.innerHTML = '';
  users.forEach(u => {
    const row = document.createElement('tr');
    const cellName = document.createElement('td');
    cellName.innerText = u.username;
    const cellStats = document.createElement('td');
    cellStats.innerText = `${u.wins} / ${u.games}`;
    const cellGold = document.createElement('td');
    cellGold.innerText = u.balance;
    row.appendChild(cellName);
    row.appendChild(cellStats);
    row.appendChild(cellGold);
    leaderboardBody.appendChild(row);
    leaderboardModal.showModal();
  })
};

btnCreateGame.addEventListener('click', () => createRoomModal.showModal());
btnFindGames.addEventListener('click', () => findGame());
btnLeaderboard.addEventListener('click', () => leaderbord());
btnCloseLeaderboard.addEventListener('click', () => leaderboardModal.close())
logoutBtn.addEventListener('click', () => logout());
btnCloseCreate.addEventListener('click', () => createRoomModal.close());
btnCloseFind.addEventListener('click', () => findRoomsModal.close());

btnConfirmCreate.addEventListener('click', async () => {
  try {
    const bet = roomBetInput.value;
    if (!bet) {
      return alert('Нужно ввести ставку');
    }
    const response = await createRoomReq(bet);
    console.log('Комната создана! ID:', response.room_id);
    createRoomModal.close();
    connectWebSocket(response.room_id);
  } catch (error) {
    console.error(`Ошибка создания комнаты: `, error);
  }
});

export const findGame = async () => {
  findRoomsModal.showModal();
  roomsListDOM.innerHTML = '';
  try {
    const roomsData = await getRooms();
    if (!roomsData || roomsData.length === 0) {
      roomsListDOM.innerText = 'Пока не создано ни одной игры - стань первым!';
      return;
    }
    roomsData.forEach(room => {
      const newRoomLi = document.createElement('li');
      newRoomLi.classList.add('rooms-list-row');
      newRoomLi.innerText = `Стол игрока ${room.creator}\n Ставка: ${room.bet_amount}`;
      const joinGameBtn = document.createElement('button');
      joinGameBtn.innerText = 'Войти';
      joinGameBtn.classList.add('btn-medieval');
    
      joinGameBtn.addEventListener('click', async () => {
        try {
          const res = await joinRoomReq(room.id);
          console.log('Успешный вход в игру! ID:', res.room_id);
          findRoomsModal.close();
          connectWebSocket(res.room_id);
        } catch (error) {
          console.error('Ошибка входа в игру: ', error)
        }
      })
      newRoomLi.appendChild(joinGameBtn);
      roomsListDOM.appendChild(newRoomLi);
    })
  } catch (error) {
    console.error('Ошибка поиска игры: ', error)
  }
};

const savedToken = localStorage.getItem('game_token');
if (savedToken) {
  console.log("Загрузка страницы: ищем активную игру...");
  connectWebSocket();
}