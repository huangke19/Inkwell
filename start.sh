#!/bin/zsh
cd "$(dirname "$0")"

if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

PID_FILE=".inkwell.pid"

if [ -f "$PID_FILE" ] && kill -0 "$(cat $PID_FILE)" 2>/dev/null; then
  echo "Inkwell 已在运行（PID $(cat $PID_FILE)），访问 http://localhost:${PORT:-8081}"
  exit 0
fi

nohup go run main.go > inkwell.log 2>&1 &
echo $! > "$PID_FILE"

sleep 2
if kill -0 "$(cat $PID_FILE)" 2>/dev/null; then
  echo "Inkwell 已启动（PID $(cat $PID_FILE)），访问 http://localhost:${PORT:-8081}"
else
  echo "启动失败，查看 inkwell.log 了解详情"
  rm -f "$PID_FILE"
  exit 1
fi
