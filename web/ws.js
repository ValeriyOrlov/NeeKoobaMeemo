import { handleGameEvent, setGameSocket } from "./game.js";
import { showScreen } from "./ui.js";

export function connectWebSocket(roomId) {
  const token = localStorage.getItem('game_token');
  if (!token) return;

  // Передаём и токен, и ID комнаты в URL
  const wsUrl = `ws://localhost:8081/ws?token=${token}&room_id=${roomId}`;
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
  
  socket.onerror = (err) => console.error('Ошибка WebSocket:', err);
  socket.onclose = () => console.log('соединение закрыто');

  return socket;
}