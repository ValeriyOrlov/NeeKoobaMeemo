import { handleGameEvent, setGameSocket } from "./game.js";
import { getValidToken, loadProfile } from "./api.js";
import { showScreen, showGameStatus, showWaitingOverlay, hideWaitingOverlay, hideReconnectingState } from "./ui.js";

const wsHost = window.ENV.WS_URL;

// Флаг для отслеживания перезагрузки/закрытия вкладки браузера
let isUnloading = false;
window.addEventListener('beforeunload', () => {
  isUnloading = true;
});

// Переменные для Expotentional Backoff
let reconnectAttempts = 0;
let reconnectTimerId = null; // Храним ID таймера для возможности отмены
const MAX_RECONNECT_DELAY = 30000; // максимум 30 секунд между попытками
const BASE_RECONNECT_DELAY = 1000; // Базовая задержка 1 секунда
let expectedRoomContext = null; // Глобальная память о том, что мы пытаемся вернуться в игру

function getReconnectDelay(attempt) {
  const delay = Math.min(BASE_RECONNECT_DELAY * Math.pow(2, attempt), MAX_RECONNECT_DELAY);
  const jitter = delay * 0.2 * (Math.random() - 0.5); // Разброс +-20%
  return Math.floor(delay + jitter);
}

export async function connectWebSocket(roomId, isJoining = false) {
  const token = await getValidToken();
  if (!token) {
    console.warn("Нет токена для WebSocket. Игрок не авторизован");
    return;
  }
// Запоминаем, что игрок находился в комнате, чтобы после обрыва связи продолжить ретраи
  if (roomId) expectedRoomContext = roomId;

  let wsUrl = `${wsHost}?token=${token}`;
  if (roomId) {
    wsUrl += `&room_id=${roomId}`;
  }
  const socket = new WebSocket(wsUrl);

  socket.onopen = () => {
    setGameSocket(socket);
    reconnectAttempts = 0; // Успешное соединение сбрасывает счётчик попыток
    hideReconnectingState(); // Убираем плашку офлайна при успешном коннекте
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
          expectedRoomContext = null; // Очищаем контекст при ручном выходе
          socket.close(1000, "User cancelled"); // Передаем корректный код штатного закрытия
          hideWaitingOverlay();
          showGameStatus("Вы покинули комнату");
        };
      }
    }
  };

  socket.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);

      // Переключаем экран только при старте или успешном восстановлении
      if (msg.type === "GAME_STARTED" || msg.type === "GAME_RESTORED") {
        expectedRoomContext = "active_game"; // Подтверждаем, что мы в активной игре
        hideWaitingOverlay();
        hideReconnectingState();
        showScreen('game');

        if (msg.type === "GAME_RESTORED") {
          showGameStatus("Соединение успешно восстановлено!");
        } else {
          showGameStatus("Игра началась!");
        }
      }

      // Обработка дисконнекта противника (если сервер присылает такое событие)
      if (msg.type === "OPPONENT_OFFLINE") {
        showOpponentOfflineWarning(() => {
          expectedRoomContext = null;
          socket.close(1000, "Left abandoned game");
          loadProfile();
        });
      }

      if (msg.type === "OPPONENT_RECONNECTED") {
        hideReconnectingState();
        showGameStatus("Противник вернулся в игру!");
      }

      // Если сервер вернул ошибку (например, игрок не в игре)
      if (msg.type === "ERROR") {
        hideWaitingOverlay();
        hideReconnectingState();
        expectedRoomContext = null;
        // Показываем сообщение об ошибке, только если подключение шло к конкретной комнате
        if (roomId || expectedRoomContext) {
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
    reconnectTimerId = setTimeout(() => connectWebSocket(null, false), delay);
    // 1. Игнорируем разрыв при перезагрузке страницы (F5) или закрытии вкладки
    if (isUnloading) {
      return;
    }

    console.warn(`[WS CLOSE] Код: ${event.code}, Причина: "${event.reason}", WasClean: ${event.wasClean}`);

// Проверяем expectedRoomContext, так как isGameActive и roomId могут быть сброшены в новой функции
    if (expectedRoomContext) {
      // 1006 - ненормальное закрытие (обрыв сети, падение сервера)
      if (event.code === 1006 || !event.wasClean) {
        const delay = getReconnectDelay(reconnectAttempts);
        reconnectAttempts++;
        
        showGameStatus(`Связь прервана. Попытка ${reconnectAttempts} через ${Math.round(delay/1000)}с...`, delay);
        
        // Рекурсивно вызываем переподключение. roomId передаем как null, 
        // так как сервер сам найдет брошенную игру по токену
        setTimeout(() => connectWebSocket(null, false), delay);
      } else {
        // Штатное закрытие соединения (например, конец игры или ручной выход)
        expectedRoomContext = null;
        loadProfile();
      }
    } else {
      console.log('Соединение закрыто (игрок в лобби)');
    }
  };

  return socket;
}
