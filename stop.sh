#!/bin/zsh
cd "$(dirname "$0")"

PID_FILE=".inkwell.pid"

if [ ! -f "$PID_FILE" ]; then
  echo "未找到 PID 文件，Inkwell 可能未在运行"
  exit 0
fi

PID=$(cat "$PID_FILE")

if kill -0 "$PID" 2>/dev/null; then
  kill "$PID"
  rm -f "$PID_FILE"
  echo "Inkwell 已停止（PID $PID）"
else
  echo "进程 $PID 不存在，清理 PID 文件"
  rm -f "$PID_FILE"
fi
