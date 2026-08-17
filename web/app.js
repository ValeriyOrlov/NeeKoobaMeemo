// инициализация web socket соединения

const server = "ws://localhost:8080/ws"
let socket;

const playerNameInput = document.querySelector('#player-name-input');
const playerNameButton = document.querySelector('#player-name-button');
const gameScreen = document.querySelector('.game-screen');
const loginScreen = document.querySelector('.login-screen');
const gameStatus = document.querySelector('.game-status');
const gameLogContainer = document.querySelector('.game-log-container');
const gameLog = document.createElement('li');
const btnRoll = document.querySelector('.btn-roll');
const btnSelect = document.querySelector('.btn-select');
const btnBank = document.querySelector('.btn-bank');
const diceContainer = document.querySelector('.dice-container');
// Элементы UI
const selfBank = document.querySelector('#self-bank');
const opponentBank = document.querySelector('#opponent-bank');
const selfSelected = document.querySelector('#self-selected');
const opponentSelected = document.querySelector('#opponent-selected');
const opponentNameEl = document.querySelector('#opponent-name');
const selfNameEl = document.querySelector('#self-name');

let opponentPlayerName = '';

const gameDisplayHandler = (hasLogin) => {
  if (hasLogin) {
    gameScreen.style.display = 'flex';
    loginScreen.style.display = 'none';
  } else {
    gameScreen.style.display = 'none'; 
    loginScreen.style.display = 'flex';
  }
}

const renderDice = (diceArray) => {
  diceContainer.innerHTML = '';
  
  if (!diceArray || diceArray.length === 0) {
    return;
  }

  diceArray.forEach(side => {
    const cube = document.createElement('div');
    cube.classList.add('cube')
    cube.innerHTML = side
    diceContainer.appendChild(cube)
    cube.addEventListener('click', () => cube.classList.toggle('selected'));
  });   
}

const updateControls = (isMyTurn, canRoll, canSelect, canBank) => {
  if (!isMyTurn) {
    btnRoll.disabled = true;
    btnSelect.disabled = true;
    btnBank.disabled = true; 
  } else {
    btnRoll.disabled = !canRoll;
    btnSelect.disabled = !canSelect;
    btnBank.disabled = !canBank;
  }
}

playerNameButton.addEventListener('click', () => {
  const playerName = playerNameInput.value;
  if (playerName.length === 0) {
    alert("Нужно ввести имя!");
    return
  }
  gameDisplayHandler(true)
  selfNameEl.innerText = playerName;
  updateControls(false, false, false, false);

  socket = new WebSocket(`${server}?name=${playerName}`)
  socket.onopen = function(event) {
    console.log('Соединение установлено.');
  }

  socket.onmessage = async function(event) {
    const parsedData = JSON.parse(event.data);
    const activePlayer = parsedData.active_player;
    const banks = parsedData.banks;
    const eventType = parsedData.type;
    const eventMsg = parsedData.message;
    const gameLog = document.createElement('li');
    const printLog = (logMessage) => {
      gameLog.textContent = logMessage;
      gameLogContainer.appendChild(gameLog);
      gameLogContainer.scrollTo({
        top: gameLogContainer.scrollHeight,
        behavior: 'smooth'
      });
    }

    // Синхронизация банков
    if (banks && opponentPlayerName) {
      selfBank.innerText = banks[playerName] || 0;
      opponentBank.innerText = banks[opponentPlayerName] || 0;
    }

    const isMyTurn = (activePlayer === playerName);

    switch (eventType) {
      case "GAME_STARTED":
        if (banks) {
          opponentPlayerName = Object.keys(banks).find(name => name !== playerName) || 'Противник';
          opponentNameEl.innerText = opponentPlayerName;
          selfBank.innerText = banks[playerName] || 0;
          opponentBank.innerText = banks[opponentPlayerName] || 0;
        }
        gameStatus.innerText = `Статус игры: ${eventMsg}`;
        // В начале игры активна только кнопка броска
        updateControls(isMyTurn, true, false, false);
        break;
      case "TURN_CHANGED":
        diceContainer.innerHTML = '';
        selfSelected.innerText = 0;
        opponentSelected.innerText = 0;
        gameStatus.innerText = `Статус игры: ${eventMsg}`;
        printLog(eventMsg);
        // Смена хода: можно бросать, откладывать и банк пока не доступны
        updateControls(isMyTurn, true, false, false);
        break;
      case "DICE_ROLLED":
        renderDice(parsedData.dice);
        printLog(`Выкатилось ${parsedData.dice}`);
        // После броска: бросать повторно нельзя, ждем выбора кубиков
        updateControls(isMyTurn, false, true, true);
        break;
      case "DICE_SELECTED":
        renderDice(parsedData.dice);
        // После отбора кубиков: можно снова бросать оставшиеся ИЛИ положить очки в банк
        updateControls(isMyTurn, true, false, true);
        printLog(`Отложено ${parsedData.score} очков!`);
        // Отрисовка отложенных очков в зависимости от того, кто ходит
        if (isMyTurn) {
          selfSelected.innerText = parsedData.score;
        } else {
          opponentSelected.innerText = parsedData.score;
        }

        break;
      case "MY_RESPECTS":
        renderDice(parsedData.dice);
        printLog(`${parsedData.message}`);
        // Обновляем очки игрока, который сделал ход
        if (isMyTurn) {
          selfSelected.innerText = parsedData.score;
        } else {
          opponentSelected.innerText = parsedData.score;
        }
        // При «Маааё почтение» открывается полный набор кубиков, можно бросать дальше или в банк
        updateControls(isMyTurn, true, false, true);
        break;
      case "ZONK":
        renderDice(parsedData.dice);
        printLog(`Выпало ${parsedData.dice}. Вы не выбросили призовых кубиков\n.${parsedData.message}`);
        selfSelected.innerText = 0;
        opponentSelected.innerText = 0;
        // При зонке ход переходит, блокируем свои кнопки
        updateControls(false, false, false, false);
        break;
      case "GAME_OVER":
        alert(parsedData.message);
        updateControls(false, false, false, false);
        break;
      case "PLAYER_LEFT":
        alert(parsedData.message);
        location.reload();
        break;
      case "ERROR":
        gameStatus.innerText = `Статус игры: ${eventMsg}`;
        break;
      case "SYSTEM":
        printLog(eventMsg);
        break;
    }
  }
  socket.onclose = function(event) {
    if (event.wasClean) {
      console.log('Соединение закрыто чисто');
    } else {
      console.log('Обрыв соединения');
    }
  }
})

const selectedDiceScore = () => {
  const selectedDice = document.querySelectorAll('.cube.selected');
  const selectedDiceNums = [];
  selectedDice.forEach(el => {
    const parsedNum = parseInt(el.innerText);
    selectedDiceNums.push(parsedNum);
  });

  return selectedDiceNums;
}

btnRoll.addEventListener('click', () => {
  const rollEvent = JSON.stringify({ type: "ROLL" });
  socket.send(rollEvent);
});

btnBank.addEventListener('click', () => {
  const selectedDiceNums = selectedDiceScore()
  const bankEvent = JSON.stringify({
    type: "BANK" ,
    dice: selectedDiceNums,
  });
  socket.send(bankEvent);
});

btnSelect.addEventListener('click', () => {
  const selectedDiceNums = selectedDiceScore();
  const selectedData = JSON.stringify({ type: "SELECT_DICE", dice: selectedDiceNums });
  socket.send(selectedData);
})