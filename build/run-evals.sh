#!/bin/sh
# Прогон tests/cases.json по всем моделям: для каждой свой llama-server, время загрузки и пик ОЗУ.
# Запускать на телефоне из ~/kuzya: ./run-evals.sh [TAG [модель ...]]. Результаты: eval-<имя><TAG>.txt/.jsonl, meta-<имя><TAG>.txt
# Набор запросов: переменная CASES (по умолчанию cases.json).
# Перед каждой моделью ждёт, пока батарея (qcom-battery) остынет ниже 45 °C. Во время прогона температуру не проверяет.
cd "$HOME/kuzya" || exit 1
TAG=$1; [ $# -gt 0 ] && shift
[ $# -eq 0 ] && set -- Qwen3.5-2B-Q4_K_M xLAM-2-1B-fc-r-Q4_K_M granite-4.1-3b-Q4_K_M gemma-4-E2B-it-Q4_K_M Qwen3.5-4B-Q4_K_M
for m in "$@"; do
  pkill llama-server; sleep 3
  while [ "$(for z in /sys/class/thermal/thermal_zone*; do [ "$(cat $z/type)" = qcom-battery ] && cat $z/temp; done)" -ge 45000 ]; do sleep 30; done
  t0=$(date +%s)
  ./bin/llama-server -m "models/$m.gguf" --jinja --chat-template-kwargs '{"enable_thinking":false}' \
    -t 4 -c 4096 --host 127.0.0.1 --port 8080 > "server-$m$TAG.log" 2>&1 &
  pid=$!
  until curl -sf http://127.0.0.1:8080/health >/dev/null; do
    kill -0 $pid 2>/dev/null || { echo "server died" > "meta-$m$TAG.txt"; continue 2; }
    sleep 1
  done
  t1=$(date +%s)
  ./bin/kuzya-eval -cases "${CASES:-cases.json}" -out "eval-$m$TAG.jsonl" > "eval-$m$TAG.txt" 2>&1
  hwm=$(grep VmHWM /proc/$pid/status)
  echo "load_seconds=$((t1-t0)) $hwm" > "meta-$m$TAG.txt"
  kill $pid; wait $pid 2>/dev/null
done
echo finished > evals-done.txt
