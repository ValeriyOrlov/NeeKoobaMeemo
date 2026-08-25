// Звуковые эффекты
const sfxRoll = new Audio('../sounds/dice.wav');
const sfxBank = new Audio('../sounds/bank.wav');
const sfxSelect = new Audio('../sounds/writingPen.wav');

let socket;
let isMyTurn = false;
let isZonkPending = false;

// === ССЫЛКИ НА ИГРОВЫЕ ЭЛЕМЕНТЫ ===
const diceContainer = document.querySelector('.dice-container');
const gameStatus = document.querySelector('.game-status');
const selfBankEl = document.getElementById('self-bank');
const selfSelectedEl = document.getElementById('self-selected');
const opponentBankEl = document.getElementById('opponent-bank');
const opponentSelectedEl = document.getElementById('opponent-selected');
const selfNameEl = document.getElementById('self-name');
const opponentNameEl = document.getElementById('opponent-name');
const profileUsername = document.getElementById('profile-username');

const gameLogContainer = document.querySelector('.game-log-container');

const btnRoll = document.querySelector('.btn-roll');
const btnBank = document.querySelector('.btn-bank');
const btnSelect = document.querySelector('.btn-select');

let selectedDiceIndices = new Set();
let currentDiceValues = [];
let myUsername = "";

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
    sfxRoll.currentTime = 0;
    sfxRoll.play();
    selectedDiceIndices.clear();
    sendAction('ROLL');
});

// Кнопка "В банк" — сохраняет очки и передает ход сопернику
btnBank.addEventListener('click', () => {
    sfxBank.currentTime = 0;
    sfxBank.play();
    
    const selectedVals = getSelectedDiceValues();
    sendAction('BANK', selectedVals);
    selectedDiceIndices.clear();
});

// Кнопка "Отложить" — записывает выделенные кубики на временный счет
btnSelect.addEventListener('click', () => {
    const selectedVals = getSelectedDiceValues();
    if (selectedVals.length === 0) {
        alert('Выберите хотя бы один призовой кубик!');
        return;
    }

    sfxSelect.currentTime = 0;
    sfxSelect.play();
    sendAction('SELECT_DICE', selectedVals);
});

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
        
        const MIN_DISTANCE = 75; 
        const placedPositions = [];

        diceArray.forEach((val, idx) => {
            const wrapper = document.createElement('div');
            wrapper.className = 'dice-wrapper';
            
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

            const maxX = diceContainer.clientWidth - 70;
            const maxY = diceContainer.clientHeight - 70;
            let randomX = 0;
            let randomY = 0;
            let hasOverlap = true;
            let attempts = 0;

            while (hasOverlap && attempts < 100) {
                randomX = Math.max(10, Math.floor(Math.random() * maxX));
                randomY = Math.max(10, Math.floor(Math.random() * maxY));

                hasOverlap = placedPositions.some(pos => {
                    const deltaX = Math.abs(pos.x - randomX);
                    const deltaY = Math.abs(pos.y - randomY);
                    return deltaX < MIN_DISTANCE && deltaY < MIN_DISTANCE;
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

            // Только подсветка по клику, без мгновенной отправки
            diceEl.addEventListener('click', () => {
                if (!isMyTurn) return;
                //sfxSelect.currentTime = 0;
                //sfxSelect.play();

                diceEl.classList.toggle('selected-3d');
                
                if (selectedDiceIndices.has(idx)) {
                    selectedDiceIndices.delete(idx);
                } else {
                    selectedDiceIndices.add(idx);
                }
            });
        });
    } else {
        // --- ОТКЛАДЫВАНИЕ (Удаление выбранных со стола) ---
        const allWrappers = diceContainer.querySelectorAll('.dice-wrapper');
        
        allWrappers.forEach(wrapper => {
            const dice = wrapper.querySelector('.dice-3d');
            if (dice.classList.contains('selected-3d')) {
                wrapper.style.transform = 'scale(0)';
                wrapper.style.opacity = '0';
                setTimeout(() => wrapper.remove(), 300);
            }
        });
        
        selectedDiceIndices.clear();
        
        // Перепривязка индексов для оставшихся кубиков
        setTimeout(() => {
            const remainingDice = diceContainer.querySelectorAll('.dice-3d');
            remainingDice.forEach((diceEl, newIdx) => {
                const newDiceEl = diceEl.cloneNode(true);
                diceEl.parentNode.replaceChild(newDiceEl, diceEl);
                
                newDiceEl.addEventListener('click', () => {
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

// === ОСНОВНОЙ ОБРАБОТЧИК СОБЫТИЙ СЕРВЕРА ===
export function handleGameEvent(msg) {
    console.log("Событие от сервера:", msg);

    if (msg.message && msg.type !== "ZONK") {
        gameStatus.innerText = msg.message;
    }

    switch (msg.type) {
        case "GAME_STARTED":
            myUsername = profileUsername ? profileUsername.innerText : "";
            selfNameEl.innerText = myUsername;
            
            const players = Object.keys(msg.banks);
            const opponent = players.find(p => p !== myUsername) || "Соперник";
            opponentNameEl.innerText = opponent;

            updateBanks(msg.banks);
            resetSelectedScores();
            printLog(msg.message || "⚔️ Игра началась!");
            toggleControls(msg.active_player === myUsername);
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
            gameStatus.innerText = '💥 Пупупууу :/ Очки раунда сгорели.';
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
                printLog(msg.message || `🔄 Ход перешел к игроку ${msg.active_player}`);
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
            gameStatus.innerText = msg.message;
            updateScores(msg.active_player, msg.score);
            toggleControls(msg.active_player === myUsername);
            printLog(msg.message);
            break;

        case "GAME_OVER":
            toggleControls(false);
            printLog(msg.message || "🏆 Игра завершена!");
            setTimeout(() => {
                alert(msg.message);
            }, 1000);
            break;

        case "ERROR":
            alert(msg.message);
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