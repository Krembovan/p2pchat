#!/usr/bin/env bash
# p2pchat — установка в одну команду
# curl -sSL https://example.com/p2pchat/install.sh | bash
set -euo pipefail

BIN="p2pchat"
DEST="${1:-.}"

# 1) Если рядом уже есть собранный бинарник — копируем
if [ -f "./build/${BIN}-linux-amd64" ]; then
  cp "./build/${BIN}-linux-amd64" "${DEST}/${BIN}"
  chmod +x "${DEST}/${BIN}"
  echo "✓ p2pchat → ${DEST}/${BIN}"
  echo "  Запуск: ${DEST}/${BIN}"
  exit 0
fi

# 2) Пробуем скачать готовый бинарник
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "✗ architecture: $ARCH"; exit 1 ;;
esac

URL="https://github.com/USER/p2pchat/releases/latest/download/${BIN}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then URL="${URL}.exe"; fi

if command -v curl &>/dev/null; then
  echo "↓ ${URL}"
  curl -sSL "$URL" -o "${DEST}/${BIN}"
elif command -v wget &>/dev/null; then
  wget -q "$URL" -O "${DEST}/${BIN}"
else
  echo "✗ нужен curl или wget"
  echo ""
  echo "Вручную: скачай https://github.com/USER/p2pchat/releases/latest"
  echo "И положи файл ${BIN}-${OS}-${ARCH} рядом с собой как ${BIN}"
  exit 1
fi

[ "$OS" != "windows" ] && chmod +x "${DEST}/${BIN}"
echo "✓ p2pchat → ${DEST}/${BIN}"
echo "  Запуск: ${DEST}/${BIN}"
