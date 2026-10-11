## 2023-10-27 - [CSWSH in Gorilla WebSocket configuration]
**Vulnerability:** A Cross-Site WebSocket Hijacking (CSWSH) vulnerability existed because `CheckOrigin` unconditionally allowed all origins by returning `true` for all requests (`github.com/gorilla/websocket` Upgrader configuration).
**Learning:** This is a common pattern for local development that accidentally got leaked to the main application configuration. It exposes authenticated websocket endpoints to CSRF-style attacks from malicious origins.
**Prevention:** Always restrict CORS/WebSocket origin checks to the environment running it. If development mode needs full permissiveness, move the `CheckOrigin: true` override to inside an explicit `DEV` mode block.
