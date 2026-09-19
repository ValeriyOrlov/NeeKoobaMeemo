import { handleGameEvent, setGameSocket } from "./game.js";
import { getValidToken, loadProfile } from "./api.js";
import { showScreen, showGameStatus, showWaitingOverlay, hideWaitingOverlay } from "./ui.js";

const wsHost = window.ENV.WS_URL;

// Флаг для отслеживания перезагрузки/закрытия вкладки браузера
let isUnloading = false;
window.addEventListener('beforeunload', () => {
  isUnloading = true;
});

export async function connectWebSocket(roomId, isJoining = false) {
  const token = await getValidToken();
  if (!token) {
    console.warn("Нет токена для WebSocket. Игрок не авторизован.");
    return;
  }

  let wsUrl = `${wsHost}?token=${token}`;
  if (roomId) {
    wsUrl += `&room_id=${roomId}`;
  }
  const socket = new WebSocket(wsUrl);

  // Флаг, указывающий, началась ли или восстановилась ли игра
  let isGameActive = false;

  socket.onopen = () => {
    setGameSocket(socket);

    if (roomId) {
      showGameStatus("Соединение установлено");

      if (isJoining) {
        showWaitingOverlay("Ожидаем согласия создателя комнаты");
      } else {
        showWaitingOverlay("Комната создана. Ожидаем второго игрока...");
      }

      const cancelBtn = document.getElementById('cancel-waiting-btn');
      if (cancelBtn) {
        cancelBtn.onclick = () => {
          socket.close();
          hideWaitingOverlay();
          showGameStatus("Вы покинули комнату");
        };
      }
    }
  };

  socket.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      console.log('Событие от сервера: ', msg);

      // Переключаем экран только при старте или успешном восстановлении
      if (msg.type === "GAME_STARTED" || msg.type === "GAME_RESTORED") {
        isGameActive = true;
        hideWaitingOverlay();
        showScreen('game');

        if (msg.type === "GAME_RESTORED") {
          showGameStatus("Соединение успешно восстановлено!");
        } else {
          showGameStatus("Игра началась!");
        }
      }

      // Если сервер вернул ошибку (например, игрок не в игре)
      if (msg.type === "ERROR") {
        hideWaitingOverlay();
        // Показываем сообщение об ошибке, только если подключение шло к конкретной комнате
        if (roomId) {
          showGameStatus(`❌ ${msg.message}`, 4000);
        }
        loadProfile();
      }

      handleGameEvent(msg);
    } catch (err) {
      console.error("Ошибка обработчика входящего сообщения:", err);
    }
  };

  socket.onerror = (err) => {
    hideWaitingOverlay();
    if (!roomId) {
      console.log('Активных игр для восстановления не найдено');
      return;
    }
    console.error('Сбой WebSocket соединения:', err);
  };

  socket.onclose = (event) => {
    hideWaitingOverlay();
    setGameSocket(null);

    // 1. Игнорируем разрыв при перезагрузке страницы (F5) или закрытии вкладки
    if (isUnloading) {
      return;
    }

    console.warn(`[WS CLOSE] Код: ${event.code}, Причина: "${event.reason}", WasClean: ${event.wasClean}`);

    // 2. Показываем ошибку только если пользователь находился в игре или подключался к конкретной комнате
    if (roomId || isGameActive) {
      if (event.code === 1006) {
        showGameStatus("❌ Потеряно соединение с сервером игры", 4000);
      } else if (!event.wasClean) {
        showGameStatus(`❌ Соединение закрыто: ${event.reason || 'Ошибка сети'}`, 4000);
      }
      loadProfile();
    } else {
      // Тихий режим для находящихся в лобби
      console.log('Соединение закрыто (игрок в лобби)');
    }
  };

  return socket;
}
