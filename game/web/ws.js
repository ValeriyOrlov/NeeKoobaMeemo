import { handleGameEvent, setGameSocket } from "./game.js";
import { getValidToken } from "./api.js";
import { showScreen } from "./ui.js";

const wsHost = window.ENV.WS_URL;

export async function connectWebSocket(roomId) {
  const token = await getValidToken();
  if (!token) {
    console.warn("Нет токена для WebSocket. Игрок не авторизован.");
    return;
  };

  // Передаём и токен, и ID комнаты в URL
  let wsUrl = `${wsHost}?token=${token}`;
  if (roomId) {
    wsUrl += `&room_id=${roomId}`;
  }
  const socket = new WebSocket(wsUrl);

  socket.onopen = () => {
    console.log(`Успешно подключились к комнате: ${roomId}`);
    // Переключаем экран с лобби на игровой стол
    setGameSocket(socket);
    showScreen('game');
  };

  socket.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    console.log('Событие от сервера: ', msg);
    // Здесь будем обрабатывать события GAME_STARTED, TURN_CHANGED и т.д
    handleGameEvent(msg);
  };
  
  socket.onerror = (err) => {
    // Ошибка ожидаема, если мы пытались восстановить соединение, но активной игры не было
    if (!roomId) {
      console.log('Активных игр для восстановления не найдено');
      return;
    }
    console.error('Ошибка WebSocket:', err);
  }
  socket.onclose = () => console.log('соединение закрыто');

  return socket;
}