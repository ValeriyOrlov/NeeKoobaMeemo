(function () {
  const host = window.location.hostname;
  const protocol = window.location.protocol; // http: или https:
  const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:';

  window.ENV = {
    // Автоматически берет хост (localhost или публичный IP)
    GAME_SERVER_URL: `${protocol}//${host}`,
    AUTH_SERVER_URL: `${protocol}//${host}/api/auth`,
    WS_URL: `${wsProtocol}//${host}/ws`
  };
})();
