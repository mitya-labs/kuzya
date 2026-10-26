#!/bin/sh
# Холодный запрос против запроса после восстановления KV-кэша слота. Вывод в bench-slot.log.
cd "$HOME/kuzya" || exit 1
M=${1:-Qwen3.5-2B-Q4_K_M}
mkdir -p slots
echo '[{"id":1,"q":"какой заряд батареи?","tool":"battery_status","args":{}}]' > cases-one.json
temp() { for z in /sys/class/thermal/thermal_zone*; do t=$(cat $z/type); [ "$t" = qcom-battery ] && b=$(cat $z/temp); [ "$t" = cpu5-thermal ] && c=$(cat $z/temp); done; echo "battery=$b cpu5=$c"; }
start() {
  pkill -x llama-server; sleep 3
  ./bin/llama-server -m "models/$M.gguf" --jinja --chat-template-kwargs '{"enable_thinking":false}' \
    -t 4 -c 4096 --host 127.0.0.1 --port 8080 --slot-save-path "$HOME/kuzya/slots" > "server-slot-$1.log" 2>&1 &
  until curl -sf http://127.0.0.1:8080/health >/dev/null; do sleep 1; done
}
{
echo "--- 1. cold start | $(temp) | $(date +%H:%M:%S)"
start 1
t0=$(date +%s); ./bin/kuzya-eval -cases cases-one.json | head -1; echo "cold seconds: $(( $(date +%s)-t0 ))"
echo "--- 2. save slot"
curl -s -X POST 'http://127.0.0.1:8080/slots/0?action=save' -H 'Content-Type: application/json' -d '{"filename":"kuzya.bin"}'; echo
ls -l slots/
echo "--- 3. restart + restore | $(temp)"
start 2
t0=$(date +%s); curl -s -X POST 'http://127.0.0.1:8080/slots/0?action=restore' -H 'Content-Type: application/json' -d '{"filename":"kuzya.bin"}'; echo; echo "restore seconds: $(( $(date +%s)-t0 ))"
t0=$(date +%s); ./bin/kuzya-eval -cases cases-one.json | head -1; echo "after-restore seconds: $(( $(date +%s)-t0 ))"
echo "--- 4. restart without restore (control) | $(temp)"
start 3
t0=$(date +%s); ./bin/kuzya-eval -cases cases-one.json | head -1; echo "cold again seconds: $(( $(date +%s)-t0 ))"
echo "--- end | $(temp)"
pkill -x llama-server
} > bench-slot.log 2>&1
echo finished > bench-slot-done.txt
