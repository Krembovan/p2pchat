# p2pchat

P2P-мессенджер. Никаких серверов — пользователи соединяются напрямую.

## Быстрый старт

**Linux/macOS (одной командой):**
```bash
curl -L https://github.com/Krembovan/p2pchat/releases/latest/download/p2pchat-linux-amd64 -o p2pchat && chmod +x p2pchat && ./p2pchat
```

**Windows:** скачай [p2pchat-windows-amd64.exe](https://github.com/Krembovan/p2pchat/releases/latest/download/p2pchat-windows-amd64.exe), запусти.

**Mac Apple Silicon:**
```bash
curl -L https://github.com/Krembovan/p2pchat/releases/latest/download/p2pchat-darwin-arm64 -o p2pchat && chmod +x p2pchat && ./p2pchat
```

Открой `http://localhost:8080` в браузере.
Соседи в локальной сети находятся автоматически.

## Сборка

```bash
bash build.sh          # build/* под все платформы
```

Бинарник ~9 MB, ноль зависимостей.

## Как это работает

- Каждый пир — TCP-сервер + клиент в одном процессе
- Авто-поиск по LAN через UDP broadcast (порт 42069)
- Веб-интерфейс встроен в бинарник (`//go:embed`)
- Сообщения в реальном времени через SSE
