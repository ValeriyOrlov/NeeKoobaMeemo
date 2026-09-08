// Музыка для лобби (один трек по кругу)
const lobbyMusic = new Audio('../sounds/lobby.mp3');
lobbyMusic.loop = true;
lobbyMusic.volume = 0.4;

// Музыка для игры (массив треков)
const gamePlaylist = [
    '../sounds/game/deuslower-medieval-citytavern-ambient-235876.mp3',
    '../sounds/game/sounds/game/geoffharvey-the-celtic-handmaiden-147078.mp3',
    '../sounds/game/sounds/game/turning_pages-pure-tavern-not-lo-fi-483972.mp3'
];
let currentGameTrackIndex = 0;
const gameMusic = new Audio(gamePlaylist[0]);
gameMusic.volume = 0.3;
1
// Глобальные состояния
let isMusicEnabled = false;
let currentMode = 'lobby'; // 'lobby' или 'game'

// Переключение треков в игре по завершению текущего
gameMusic.addEventListener('ended', () => {
    currentGameTrackIndex = (currentGameTrackIndex + 1) % gamePlaylist.length;
    gameMusic.src = gamePlaylist[currentGameTrackIndex];
    if (isMusicEnabled) gameMusic.play();
});

export function toggleMusic() {
    isMusicEnabled = !isMusicEnabled;
    if (isMusicEnabled) {
        playCurrentMode();
    } else {
        stopAll();
    }
    return isMusicEnabled; 
}

export function setMusicMode(mode) {
    if (currentMode === mode) return; 
    currentMode = mode;
    
    if (isMusicEnabled) {
        stopAll();
        playCurrentMode();
    }
}

function playCurrentMode() {
    if (currentMode === 'lobby') {
        lobbyMusic.play().catch(e => console.warn("Автоплей заблокирован:", e));
    } else if (currentMode === 'game') {
        gameMusic.play().catch(e => console.warn("Автоплей заблокирован:", e));
    }
}

function stopAll() {
    lobbyMusic.pause();
    gameMusic.pause();
}