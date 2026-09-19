(function () {
  const { protocol, hostname, port, host } = window.location;
  const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:';

  // Если порт 8081 — фронтенд открыт напрямую через Go-сервер (локальная разработка)
  const isLocalDev = port === '8081';

  if (isLocalDev) {
    window.ENV = {
      GAME_SERVER_URL: `${protocol}//${hostname}:8081`,
      AUTH_SERVER_URL: `${protocol}//${hostname}:8080`,
      WS_URL: `${wsProtocol}//${hostname}:8081/ws`
    };
  } else {
    // Режим с Nginx (все запросы идут на порт 80/443 через единый host)
    window.ENV = {
      GAME_SERVER_URL: `${protocol}//${host}`,
      AUTH_SERVER_URL: `${protocol}//${host}/api/auth`,
      WS_URL: `${wsProtocol}//${host}/ws`
    };
  }
})();