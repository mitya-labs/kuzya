#!/bin/sh
# Замер вариантов тюнинга CPU на одной модели: ./bench-tune.sh [файл модели]. Вывод в bench-tune.log.
# Ждёт остывания батареи ниже 40 °C, пишет температуру перед каждым вариантом.
# Baseline повторяется в конце, чтобы увидеть дрейф из-за нагрева.
cd "$HOME/kuzya" || exit 1
M=${1:-models/Qwen3.5-2B-Q4_K_M.gguf}
temp() { for z in /sys/class/thermal/thermal_zone*; do t=$(cat $z/type); [ "$t" = qcom-battery ] && b=$(cat $z/temp); [ "$t" = cpu5-thermal ] && c=$(cat $z/temp); done; echo "battery=$b cpu5=$c"; }
batt() { for z in /sys/class/thermal/thermal_zone*; do [ "$(cat $z/type)" = qcom-battery ] && cat $z/temp; done; }
while [ "$(batt)" -ge 40000 ]; do sleep 30; done
run() { echo "=== $1 | $(temp) | $(date +%H:%M)"; shift; ./bin/llama-bench -m "$M" -p 128 -n 32 -r 3 "$@" 2>&1 | grep -E "^\|"; }
{
run "A baseline -t 4" -t 4
run "B big cores strict mask 0xf0" -t 4 -C 0xf0 --cpu-strict 1
run "C flash attn on" -t 4 -fa on
run "D flash attn on + KV q8_0" -t 4 -fa on -ctk q8_0 -ctv q8_0
run "A2 baseline again" -t 4
echo "=== end | $(temp) | $(date +%H:%M)"
} > bench-tune.log 2>&1
echo finished > bench-tune-done.txt
