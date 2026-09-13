import { loadProfile } from "./api.js";
import { showScreen, setIsGameActive, hideWaitingOverlay } from "./ui.js";
import { sfx } from './sfx.js';

window.addEventListener('DOMContentLoaded', () => {
    // Включаем WebAudio на первый клик/тач в документе
    const unlockAudio = () => {
        sfx.init();
        
        // Загружаем эффекты
        sfx.loadSound('roll', '../sounds/dice.wav');
        sfx.loadSound('bank', '../sounds/bank.wav');
        sfx.loadSound('select', '../sounds/writingPen.wav');

        document.removeEventListener('pointerdown', unlockAudio);
    };

    document.addEventListener('pointerdown', unlockAudio);
});
// Звуковые эффекты
/*const sfxRoll = new Audio('../sounds/dice.wav');
const sfxBank = new Audio('../sounds/bank.wav');
const sfxSelect = new Audio('../sounds/writingPen.wav');
*/
let socket;
let isMyTurn = false;
let isZonkPending = false;

let statusTimeout = null;

// Функция для временного показа сообщения
function showGameStatus(text, duration = 3000) {
    if (!text) return;

    // Сбрасываем предыдущий таймер, если события идут подряд
    if (statusTimeout) {
        clearTimeout(statusTimeout);
    }

    gameStatus.innerText = text;
    gameStatus.classList.add('visible');

    // Скрываем через указанное время (по умолчанию 3 секунды)
    statusTimeout = setTimeout(() => {
        gameStatus.classList.remove('visible');
    }, duration);
}

// === ССЫЛКИ НА ИГРОВЫЕ ЭЛЕМЕНТЫ ===
const diceContainer = document.querySelector('.dice-container');
const gameStatus = document.querySelector('.game-status');
const selfBankEl = document.getElementById('self-bank');
const selfSelectedEl = document.getElementById('self-selected');
const opponentBankEl = document.getElementById('opponent-bank');
const opponentSelectedEl = document.getElementById('opponent-selected');
const selfNameEl = document.getElementById('self-name');
const opponentNameEl = document.getElementById('opponent-name');

const gameLogContainer = document.querySelector('.game-log-container');

const btnRoll = document.querySelector('.btn-roll');
const btnBank = document.querySelector('.btn-bank');
const btnSelect = document.querySelector('.btn-select');

const avatarButton = document.querySelector('.player-avatar-btn');
const btnHello = document.querySelector('.hello-btn');
const btnThreat = document.querySelector('.threat-btn');
const btnHurryUp = document.querySelector('.hurry-up-btn');
const wowBtn = document.querySelector('.wow-btn');
const btnSurrender = document.querySelector('.surrender-btn');

const btnCheatSheet = document.querySelector('.cheat-sheet');
const cheatSheetModal = document.getElementById('cheat-sheet-modal');
const btnCloseCheatSheetModal = document.querySelector('.btn-close-cheat-sheet-modal');

const gameoverModal = document.getElementById("gameover-modal");
const gameoverModalMsg = document.querySelector(".gameover-modal-msg");
const gameoverModalCloseBtn = document.querySelector(".btn-close-gameover-modal");

const surrenderModal = document.getElementById("surrender-modal");
const surrenderModalMsg = document.querySelector(".surrender-modal-msg");
const surrenderBtn = document.querySelector(".btn-surrender");
const surrenderModalCloseBtn = document.querySelector(".btn-close-surrender-modal")

const joinRequestModal = document.getElementById('join-request-modal');
const joinRequestMsg = document.getElementById('join-request-msg');
const btnAcceptJoin = document.getElementById('btn-accept-join');
const btnRejectJoin = document.getElementById('btn-reject-join');
const joinRejectedModal = document.getElementById('join-rejected-modal');
const btnCloseRejected = document.getElementById('btn-close-rejected');

btnAcceptJoin.addEventListener('click', () => {
    sendAction('ACCEPT_JOIN');
    joinRequestModal.close();
});

btnRejectJoin.addEventListener('click', () => {
    sendAction('REJECT_JOIN');
    joinRequestModal.close();
});

btnCloseRejected.addEventListener('click', () => {
    joinRejectedModal.close();
    loadProfile(); // Возврат в лобби
});

let selectedDiceIndices = new Set();
let currentDiceValues = [];
let myUsername = "";

function getUsernameFromToken() {
    const token = localStorage.getItem('game_token');
    if (!token) return "Путник";
    try {
        // Декодирование Base64 с поддержкой кириллицы
        const payload = JSON.parse(decodeURIComponent(escape(atob(token.split('.')[1]))));
        return payload.username || "Путник";
    } catch (e) {
        console.error("Ошибка декодирования токена:", e);
        return "Путник";
    }
}

export function setGameSocket(ws) {
    socket = ws;
}

// Вспомогательная функция отправки экшенов на сервер
function sendAction(actionType, dice = []) {
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: actionType, dice: dice }));
    }
}

const printLog = (logMessage) => {
    // 1. Поиск контейнера (если класс .game-log-container не найден, ищем по ID)
    const container = gameLogContainer || document.querySelector('.game-log-container') || document.getElementById('game-log');
    
    if (!container) {
        console.warn("Контейнер лога не найден в DOM! Проверьте класс .game-log-container в HTML.");
        return;
    }

    // 2. Извлечение текста (из строки или из поля message / Message у объекта)
    let text = "";
    if (typeof logMessage === 'string') {
        text = logMessage;
    } else if (typeof logMessage === 'object' && logMessage !== null) {
        text = logMessage.message || logMessage.Message || "";
    }

    if (!text) return;

    // 3. Добавление записи в лог
    const gameLog = document.createElement('li');
    gameLog.textContent = text;
    container.appendChild(gameLog);

    // 4. Прокрутка списка вниз к новому сообщению
    container.scrollTop = container.scrollHeight;
};

function resetSelectedScores() {
    selfSelectedEl.innerText = "0";
    opponentSelectedEl.innerText = "0";
}

// Получение значений выделенных кубиков по их индексам
function getSelectedDiceValues() {
    return Array.from(selectedDiceIndices).map(idx => currentDiceValues[idx]);
}

// === НАЖАТИЯ НА КНОПКИ УПРАВЛЕНИЯ ===
btnRoll.addEventListener('click', () => {
   /* sfxRoll.currentTime = 0;
    sfxRoll.play();*/
    sfx.play('roll');
    selectedDiceIndices.clear();
    sendAction('ROLL');
});

// Кнопка "В банк" — сохраняет очки и передает ход сопернику
btnBank.addEventListener('click', () => {
   /* sfxBank.currentTime = 0;
    sfxBank.play();*/
    sfx.play('bank');
    const selectedVals = getSelectedDiceValues();
    sendAction('BANK', selectedVals);
    selectedDiceIndices.clear();
});

// Кнопка "Отложить" — записывает выделенные кубики на временный счет
btnSelect.addEventListener('click', () => {
    const selectedVals = getSelectedDiceValues();
    if (selectedVals.length === 0) {
        const warningMsg = "Выберите хотя бы один призовой кубик!";
        showGameStatus(warningMsg);
        printLog(warningMsg);
        return;
    }
    /*	
    sfxSelect.currentTime = 0;
    sfxSelect.play();*/
    sfx.play('select');	
    sendAction('SELECT_DICE', selectedVals);
});

// КНОПКИ МЕНЮ ПОЛЬЗОВАТЕЛЯ
avatarButton.addEventListener('click', () => toggleAvatarMenu());
btnHello.addEventListener('click', () => sendChat('Привет!'));
btnHurryUp.addEventListener('click', () => sendChat('Ну чё ты..?'));
btnThreat.addEventListener('click', () => sendChat('Прощайся с золотишком!'));
wowBtn.addEventListener('click', () => sendChat('Вот эт каэшн дааа'));
btnSurrender.addEventListener('click', () => {
    surrenderModalMsg.textContent = "Вы уверены, что хотите сдаться?\n Вам засчитается поражение.";
    surrenderModal.showModal();
});
btnCheatSheet.addEventListener('click', () => cheatSheetModal.showModal());
btnCloseCheatSheetModal.addEventListener('click', () => cheatSheetModal.close());

// Кнопка закрытия модального окна конца игры
gameoverModalCloseBtn.addEventListener('click', () => {
    gameoverModal.close();
    loadProfile();
})

surrenderBtn.addEventListener('click', () => surrender());
surrenderModalCloseBtn.addEventListener('click', () => surrenderModal.close());
// Словарь вращений 3D-граней
const diceRotations = {
    1: { x: 0, y: 0 },
    2: { x: 90, y: 0 },
    3: { x: 0, y: -90 },
    4: { x: 0, y: 90 },
    5: { x: -90, y: 0 },
    6: { x: 180, y: 0 }
};

// === РЕНДЕР КУБИКОВ И КЛИКИ ПО НИМ ===
function renderDice(diceArray, isNewRoll = true) {
    currentDiceValues = diceArray || [];

    if (isNewRoll) {
        // --- НОВЫЙ БРОСОК ---
        diceContainer.innerHTML = '';
        selectedDiceIndices.clear();
        
        const placedPositions = [];

        const containerWidth = diceContainer.clientWidth || 300;
        const containerHeight = diceContainer.clientHeight || 300;

        const maxX = Math.max(10, containerWidth - 80);
        const maxY = Math.max(10, containerHeight - 80);

        diceArray.forEach((val, idx) => {
            const wrapper = document.createElement('div');
            wrapper.className = 'dice-wrapper';
            wrapper.dataset.value = val; // Сохраняем значение кубика в dataset
            
            const diceEl = document.createElement('div');
            diceEl.className = 'dice-3d';
            
            const facesConfig = [
                { class: 'front face-1', val: 1 },
                { class: 'bottom face-2', val: 2 },
                { class: 'right face-3', val: 3 },
                { class: 'left face-4', val: 4 },
                { class: 'top face-5', val: 5 },
                { class: 'back face-6', val: 6 }
            ];
            
            facesConfig.forEach(face => {
                const faceEl = document.createElement('div');
                faceEl.className = `face ${face.class}`;
                for(let i = 0; i < face.val; i++) {
                    const dot = document.createElement('div');
                    dot.className = 'dot';
                    faceEl.appendChild(dot);
                }
                diceEl.appendChild(faceEl);
            });

            wrapper.appendChild(diceEl);
            diceContainer.appendChild(wrapper);

            let randomX = 0;
            let randomY = 0;
            let hasOverlap = true;
            let attempts = 0;
            let currentMinDist = 85; 

            while (hasOverlap && attempts < 300) {
                randomX = Math.max(10, Math.floor(Math.random() * maxX));
                randomY = Math.max(10, Math.floor(Math.random() * maxY));

                if (attempts === 100) currentMinDist = 70;
                if (attempts === 200) currentMinDist = 55;

                hasOverlap = placedPositions.some(pos => {
                    const deltaX = pos.x - randomX;
                    const deltaY = pos.y - randomY;
                    return Math.hypot(deltaX, deltaY) < currentMinDist;
                });
                
                attempts++;
            }

            placedPositions.push({ x: randomX, y: randomY });
            wrapper.style.left = `${randomX}px`;
            wrapper.style.top = `${randomY}px`;

            const targetRot = diceRotations[val];
            const extraSpinsX = (Math.floor(Math.random() * 3) + 2) * 360;
            const extraSpinsY = (Math.floor(Math.random() * 3) + 2) * 360;

            setTimeout(() => {
                diceEl.style.transform = `rotateX(${targetRot.x + extraSpinsX}deg) rotateY(${targetRot.y + extraSpinsY}deg)`;
            }, 50);

            diceEl.addEventListener('pointerdown', (e) => {
                e.preventDefault(); // Защита от эмуляции двойного клика браузером
                if (!isMyTurn) return;
                
                diceEl.classList.toggle('selected-3d');
                
                if (selectedDiceIndices.has(idx)) {
                    selectedDiceIndices.delete(idx);
                } else {
                    selectedDiceIndices.add(idx);
                }
            });
        });
    } else {
        // --- ОТКЛАДЫВАНИЕ (Удаляем со стола ТОЛЬКО отложенные кубики) ---
        const selectedValues = [...diceArray];
        const allWrappers = diceContainer.querySelectorAll('.dice-wrapper');
        
        allWrappers.forEach(wrapper => {
            const val = Number(wrapper.dataset.value);
            const matchIndex = selectedValues.indexOf(val);
            
            // Если значение кубика совпадает с одним из отложенных — анимируем и убираем его
            if (matchIndex !== -1) {
                wrapper.style.transform = 'scale(0)';
                wrapper.style.opacity = '0';
                setTimeout(() => wrapper.remove(), 300);
                
                // Удаляем найденный элемент из копии массива, чтобы не удалить дубликаты повторно
                selectedValues.splice(matchIndex, 1);
            }
        });
        
        selectedDiceIndices.clear();
        
        // Актуализация значений и перепривязка событий для оставшихся на столе кубиков
        setTimeout(() => {
            const remainingWrappers = diceContainer.querySelectorAll('.dice-wrapper');
            currentDiceValues = Array.from(remainingWrappers).map(w => Number(w.dataset.value));

            remainingWrappers.forEach((wrapper, newIdx) => {
                const diceEl = wrapper.querySelector('.dice-3d');
                if (!diceEl) return;

                const newDiceEl = diceEl.cloneNode(true);
                diceEl.parentNode.replaceChild(newDiceEl, diceEl);
                
                // Замена click на pointerdown
                newDiceEl.addEventListener('pointerdown', (e) => {
                    e.preventDefault();
                    if (!isMyTurn) return;
                    sfxSelect.currentTime = 0;
                    sfxSelect.play();
                    
                    newDiceEl.classList.toggle('selected-3d');
                    if (selectedDiceIndices.has(newIdx)) {
                        selectedDiceIndices.delete(newIdx);
                    } else {
                        selectedDiceIndices.add(newIdx);
                    }
                });
            });
        }, 310);
    }
}

function toggleAvatarMenu() {
    const menu = document.getElementById("avatar-menu");
    menu.classList.toggle("hidden");
}

// Отправка реплики в чат
function sendChat(text) {
    const msg = {
        type: "CHAT",
        message: text
    };
    socket.send(JSON.stringify(msg));
    toggleAvatarMenu(); // Скрываем меню после отправки
}

// Отправка сигнала о сдаче
function surrender() {
    const msg = { type: "SURRENDER" };
        socket.send(JSON.stringify(msg));
        toggleAvatarMenu();
        surrenderModal.close();
}

// функция обновления аватаров
function setPlayerAvatars(avatarsMap) {
    if (!avatarsMap) return;

    // Наш аватар
    const myAvatar = avatarsMap[myUsername] || 'fat_cat';
    document.getElementById('self-avatar-img').src = `../pictures/avatars/${myAvatar}.jpg`;

    // Аватар соперника
    for (let player in avatarsMap) {
        if (player !== myUsername) {
            const oppAvatar = avatarsMap[player] || 'prophet';
            document.getElementById('opponent-avatar-img').src = `../pictures/avatars/${oppAvatar}.jpg`;
        }
    }
}

let turnInterval;
const TURN_DURATION = 90; // 90 секунд

function startVisualTimer() {
    clearInterval(turnInterval);
    const timerDisplay = document.getElementById("timer-display");
    
    // Фиксируем абсолютное время завершения хода
    const endTime = Date.now() + TURN_DURATION * 1000;
    
    timerDisplay.innerText = `Время на ход: ${TURN_DURATION}с`;

    turnInterval = setInterval(() => {
        // Высчитываем реальный остаток времени
        const timeLeft = Math.round((endTime - Date.now()) / 1000);
        
        if (timeLeft <= 0) {
            clearInterval(turnInterval);
            timerDisplay.innerText = "Ожидание сервера...";
        } else {
            timerDisplay.innerText = `Время на ход: ${timeLeft}с`;
        }
    }, 1000);
}

function stopVisualTimer() {
    clearInterval(turnInterval);
    document.getElementById("timer-display").innerText = "";
}

// === ОСНОВНОЙ ОБРАБОТЧИК СОБЫТИЙ СЕРВЕРА ===
export function handleGameEvent(msg) {
    console.log("Событие от сервера:", msg);
    if (msg.message && msg.type !== "ZONK") {
        showGameStatus(msg.message);
    }

    switch (msg.type) {
        case "GAME_STARTED":
            setIsGameActive(true);
            myUsername = getUsernameFromToken();
            
            const players = Object.keys(msg.banks);
            const opponent = players.find(p => p !== myUsername) || "Соперник";
            opponentNameEl.innerText = opponent;

            updateBanks(msg.banks);
            resetSelectedScores();
            printLog(msg.message || "⚔️ Игра началась!");
            toggleControls(msg.active_player === myUsername);
            startVisualTimer();
            setPlayerAvatars(msg.avatars);
            break;

        case "DICE_ROLLED":
            renderDice(msg.dice, true);
            updateScores(msg.active_player, msg.score);
            toggleControls(msg.active_player === myUsername);
            printLog(msg.message || `🎲 Игрок ${msg.active_player} бросил кубики: [${msg.dice.join(', ')}]`);
            break;

        case "DICE_SELECTED":
            renderDice(msg.dice, false);
            updateScores(msg.active_player, msg.score);
            printLog(msg.message || `🎯 Игрок ${msg.active_player} отложил кубики (очков в раунде: ${msg.score ?? 0})`);
            break;

        case "ZONK":
            isZonkPending = true;
            renderDice(msg.dice, true);
            showGameStatus('💥 Пупупууу :/ Очки раунда сгорели.');
            toggleControls(false);
            printLog(`🎲 Игрок ${msg.active_player} бросил кубики: [${msg.dice.join(', ')}]`);
            printLog(`💥 Пупупууу :/ У игрока ${msg.active_player} не выпало призовых костей.`);
            break;

        case "TURN_CHANGED":
const processTurnChange = () => {
                selectedDiceIndices.clear();
                diceContainer.innerHTML = '';
                updateBanks(msg.banks);
                resetSelectedScores();
                toggleControls(msg.active_player === myUsername);
                
                // Выводим сообщение в логи и всплывающее уведомление
                const statusMsg = msg.message || `🔄 Ход перешел к игроку ${msg.active_player}`;
                printLog(statusMsg);
                showGameStatus(statusMsg);
                
                startVisualTimer();
            };

            if (isZonkPending) {
                isZonkPending = false;
                setTimeout(processTurnChange, 2500);
            } else {
                processTurnChange();
            }
            break;

        case "MY_RESPECTS":
            selectedDiceIndices.clear();
            diceContainer.innerHTML = ''; 
            showGameStatus(msg.message);
            updateScores(msg.active_player, msg.score);
            toggleControls(msg.active_player === myUsername);
            printLog(msg.message);
            break;

        case "SYSTEM":
            printLog(msg.message);
            // Восстанавливаем элементы управления. 
            // Переменная isMyTurn "помнит" статус до обрыва соединения.
            toggleControls(isMyTurn);
            break;

        case "GAME_OVER":
            setIsGameActive(false);
            stopVisualTimer();
            toggleControls(false);
	    diceContainer.innerHTML = '';
	    gameLogContainer.innerHTML = '';
            printLog(msg.message || "🏆 Игра завершена!");
            gameoverModalMsg.textContent = msg.message;
            gameoverModal.showModal();
            break;

        case "CHAT":
            printLog(`${msg.active_player}:${msg.message}`);
            break;

        case "PLAYER_DISCONNECTED":
            printLog(msg.message);
            showGameStatus(`Ожидание... (${msg.active_player} отключился)`);
            gameStatus.classList.remove('my-turn');
            
            // Блокируем кнопки, чтобы противник ничего не нажал, пока тот переподключается
            btnRoll.disabled = true;
            btnBank.disabled = true;
            btnSelect.disabled = true;
            break;
        
        case "GAME_RESTORED":
            // Переключаем экраны
            setIsGameActive(true);
            showScreen('game');

            // 1. Восстанавливаем имена игроков
            myUsername = getUsernameFromToken();
            selfNameEl.innerText = myUsername;
            
            const allPlayers = Object.keys(msg.banks);
            const op = allPlayers.find(p => p !== myUsername) || "Соперник";
            opponentNameEl.innerText = op;
            // 2. Восстанавливаем банки и очки на столе
            updateBanks(msg.banks);
            updateScores(msg.active_player, msg.score);
            
            // 3. Восстанавливаем кубики, если бросок был уже сделан
            if (msg.dice && msg.dice.length > 0) {
                renderDice(msg.dice, true);
            } else {
                diceContainer.innerHTML = '';
                selectedDiceIndices.clear();
            }
            
            // 4. Восстанавливаем управление
            toggleControls(msg.active_player === myUsername);
            showGameStatus(msg.message);
            printLog(msg.message);
            
            // 5. Перезапускаем визуал таймера хода
            startVisualTimer(); 
            setPlayerAvatars(msg.avatars);
            break;

	case "JOIN_REQUEST":
            joinRequestMsg.innerText = `К вам подключается игрок ${msg.message}. Впустить или отклонить?`;
            joinRequestModal.showModal();
            break;

        case "JOIN_REJECTED":
            hideWaitingOverlay();
            joinRejectedModal.showModal();
            break;

        case "ERROR":
            // 1. Снимаем визуальное выделение со всех кубиков
            diceContainer.querySelectorAll('.dice-3d.selected-3d').forEach(diceEl => {
                diceEl.classList.remove('selected-3d');
            });
            
            // 2. Очищаем Set выбранных индексов
            selectedDiceIndices.clear();

            // 3. Выводим статусное сообщение и логируем ошибку
            showGameStatus(msg.message || "Ошибка хода!", 4000);
            printLog(`⚠️ ${msg.message || "Некорректное действие"}`);
            break;
    }
}

function toggleControls(turnState) {
    isMyTurn = turnState;
    btnRoll.disabled = !isMyTurn;
    btnBank.disabled = !isMyTurn;
    btnSelect.disabled = !isMyTurn;
    if (isMyTurn) {
        gameStatus.classList.add('my-turn');
    } else {
        gameStatus.classList.remove('my-turn');
    }
}

function updateBanks(banksMap) {
    if (!banksMap) return;
    if (banksMap[myUsername] !== undefined) {
        selfBankEl.innerText = banksMap[myUsername];
    }
    for (let p in banksMap) {
        if (p !== myUsername) {
            opponentBankEl.innerText = banksMap[p];
        }
    }
}

function updateScores(activePlayer, score) {
  const safeScore = score ?? 0;
    if (activePlayer === myUsername) {
        selfSelectedEl.innerText = safeScore;
    } else {
        opponentSelectedEl.innerText = safeScore;
    }
}
