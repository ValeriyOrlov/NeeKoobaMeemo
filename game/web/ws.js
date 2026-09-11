import { handleGameEvent, setGameSocket } from "./game.js";
import { getValidToken, loadProfile } from "./api.js";
import { showScreen, showGameStatus, showWaitingOverlay, hideWaitingOverlay } from "./ui.js";

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
    console.log(`WebSocket соединение установлено с комнатой: ${roomId || 'поиск'}`);    // Переключаем экран с лобби на игровой стол
    setGameSocket(socket);

    if (roomId) {
      showWaitingOverlay("Комната создана. Ожидаем второго игрока...");

      // Привязываем отмену к зарытию сокета
      const cancelBtn = document.getElementById('cancel-waiting-btn');
      if (cancelBtn) {
        cancelBtn.onclick = () => {
          socket.close();
          hideWaitingOverlay();
          showGameStatus("Вы покинули комнатy");
        };
      }
    }
  };

  socket.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      console.log('Событие от сервера: ', msg);

      // Переключаем экран на игровой стол только при старте или успешном восстановлении игры
      if (msg.type === "GAME_STARTED" || msg.type === "GAME_RESTORED") {
        hideWaitingOverlay();
        showScreen('game');
      }

      // При получении ошибок от сервера обновляем данные баланса пользователя
      if (msg.type === "ERROR") {
        hideWaitingOverlay();
        showGameStatus(`❌ ${msg.message}`, 4000);
        loadProfile();
      }

      handleGameEvent(msg);
    } catch (err) {
      console.error("Ошибка обработчика входящего сообщения:", err);
    }
  };
  
  socket.onerror = (err) => {
    hideWaitingOverlay();
    // Ошибка ожидаема, если мы пытались восстановить соединение, но активной игры не было
    if (!roomId) {
      console.log('Активных игр для восстановления не найдено');
      return;
    }
    console.error('Сбой WebSocket соединения:', err);
    showGameStatus("❌ Ошибка соединения с игровой комнатой");
    loadProfile();
  }
  socket.onclose = (event) => {
    hideWaitingOverlay();
    console.log('соединение закрыто:', event.reason || 'Завершено');
    setGameSocket(null);

    // Если закрытие нештатное - синхронизируем баланс пользователя с БД
    if (!event.wasClean) {
      loadProfile();
    }
  };

  return socket;
}