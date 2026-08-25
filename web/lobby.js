import { showScreen } from "./ui.js";
import { getLeaderboard, getRooms, createRoomReq,  joinRoomReq} from "./api.js";
import { connectWebSocket } from "./ws.js";

const btnCreateGame = document.getElementById('btn-create-game');
const btnFindGames = document.getElementById('btn-find-games');
const btnLeaderboard = document.getElementById('btn-leaderboard');
const logoutBtn = document.getElementById('btn-logout');

const leaderboardModal = document.getElementById('leaderboard-modal');
const leaderboardBody = document.getElementById('leaderboard-body');
const btnCloseLeaderboard = document.getElementById('btn-close-leaderboard');

const createRoomModal = document.getElementById('create-room-modal');
const findRoomsModal = document.getElementById('find-rooms-modal');
const roomsListDOM = document.getElementById('rooms-list');
const roomBetInput = document.getElementById('room-bet-input');
const btnConfirmCreate = document.getElementById('btn-confirm-create');
const btnCloseCreate = document.getElementById('btn-close-create')
const btnCloseFind = document.getElementById('btn-close-find');

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
      newRoomLi.innerText = `${room.creator}\n Ставка: ${room.bet_amount}`;
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
