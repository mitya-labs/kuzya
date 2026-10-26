#!/bin/sh
# Gemma: новый промпт (agent.System) на cases-all.json, затем старый (system-v2.txt) на cases-held.json. Вывод eval-ab-*.txt/.jsonl, meta-ab.txt
cd "$HOME/kuzya" || exit 1
M=gemma-4-E2B-it-Q4_K_M
temp() { for z in /sys/class/thermal/thermal_zone*; do t=$(cat $z/type); [ "$t" = qcom-battery ] && b=$(cat $z/temp); [ "$t" = cpu5-thermal ] && c=$(cat $z/temp); done; echo "battery=$b cpu5=$c"; }
pkill -x llama-server; sleep 3
./bin/llama-server -m "models/$M.gguf" --jinja --chat-template-kwargs '{"enable_thinking":false}' -t 4 -c 4096 --host 127.0.0.1 --port 8080 > server-ab.log 2>&1 &
pid=$!
until curl -sf http://127.0.0.1:8080/health >/dev/null; do sleep 1; done
{
echo "start $(temp) $(date +%H:%M)"
./bin/kuzya-eval -cases cases-all.json -out eval-ab-v3-all.jsonl > eval-ab-v3-all.txt 2>&1
echo "after v3 all $(temp) $(date +%H:%M)"
./bin/kuzya-eval -cases cases-held.json -system system-v2.txt -out eval-ab-v2-held.jsonl > eval-ab-v2-held.txt 2>&1
echo "after v2 held $(temp) $(date +%H:%M) $(grep VmHWM /proc/$pid/status)"
} > meta-ab.txt 2>&1
kill $pid
echo finished > evals-done.txt
