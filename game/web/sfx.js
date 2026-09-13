class SoundManager {
    constructor() {
        this.ctx = null;
        this.buffers = new Map();
    }

    // Инициализация контекста (должна вызываться по клику пользователя)
    init() {
        if (!this.ctx) {
            const AudioContext = window.AudioContext || window.webkitAudioContext;
            this.ctx = new AudioContext();
        }
        if (this.ctx.state === 'suspended') {
            this.ctx.resume();
        }
    }

    // Предзагрузка звукового файла в память
    async loadSound(name, url) {
        try {
            const response = await fetch(url);
            const arrayBuffer = await response.arrayBuffer();
            // Раздекодируем аудиоданные
            const audioBuffer = await new Promise((resolve, reject) => {
                this.ctx.decodeAudioData(arrayBuffer, resolve, reject);
            });
            this.buffers.set(name, audioBuffer);
        } catch (e) {
            console.error(`Ошибка загрузки звука ${name}:`, e);
        }
    }

    // Мгновенное воспроизведение
    play(name) {
        if (!this.ctx || this.ctx.state === 'suspended') {
            this.init();
        }

        const buffer = this.buffers.get(name);
        if (!buffer) return;

        const source = this.ctx.createBufferSource();
        source.buffer = buffer;
        source.connect(this.ctx.destination);
        source.start(0);
    }
}

export const sfx = new SoundManager();
