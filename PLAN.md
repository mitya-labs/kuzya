# План Кузи

Цель: офлайн-агент на Poco F1 (SD845, 6/8 ГБ ОЗУ) под skladenOS, настраивающий систему через белый список инструментов.

## Этап 0 — доступ и среда
- [x] SSH на телефон по ключу работает (Wi-Fi; проверен вход без пароля; адрес и пользователь в `LOCAL.md`).
- [x] Отключить вход по паролю (сделано пользователем).
- Состояние телефона: aarch64, 8 ядер, ОЗУ 5,5 ГБ (свободно ~4,6), диск 112 ГБ.
- [x] nosleep включён (пользователем).
- [x] Проверено 21:20 (`logs/phone-env-check.log`): aarch64, 8 ядер, ОЗУ 5,4 ГБ (доступно 4,5), swap 2,7 ГБ, диск 102 ГБ свободно.
- Компиляторов и сборочных инструментов на телефоне нет (gcc, g++, clang, cmake, make, git) — собираем в Docker на Маке (`build/Dockerfile`), на телефоне не нужны.

## Этап 1 — инференс на телефоне
- [x] Собрать `llama.cpp` (CPU, NEON) в Docker arm64/Debian sid (commit `6050b46`, `armv8.2-a+fp16`: у SD845 нет dotprod); бинарники в `~/kuzya/bin` на телефоне, `llama-server --version` запускается.
- [x] Сборка грузит Qwen3.5 (проверено на 2B и 4B); [x] Gemma 4 E2B тоже грузится; [x] сервер с `--jinja` на телефоне запущен, Qwen3.5-2B вызвала инструменты (`logs/first-run.log`).
- Скачать на Мак и перенести модели из короткого списка (скачаны Qwen3.5-4B и 2B, копируются на телефон; остальные — позже) (`docs/RESEARCH-models.md`):
  Qwen3.5-4B, Qwen3.5-2B, Gemma 4 E2B, Granite 4.1 3B, Llama 3.2 3B (все Q4_K_M); по желанию xLAM-2-1b-fc-r.
- Замерить: токенов/с, время загрузки, пик ОЗУ, влияние на батарею. Записать в `logs/`. Сделано частично (`logs/bench-qwen35.log`): Qwen3.5-2B 6,7 ток/с генерация, 12 ток/с промпт; 4B 3,0 и 4,4. Также (`logs/bench-models2-unplugged.log`, без зарядки): xLAM-2-1b 8,7 / 13,1, Gemma 4 E2B 5,6 / 8,5, Granite 3B 4,0 / 5,3 (при зарядке по USB те же цифры ±3%). Не замерены: время загрузки, пик ОЗУ, Llama 3.2 3B.
- Критерий: ≥ 3 ток/с на выбранной модели, система остаётся отзывчивой.

### Stage 1b — CPU tuning and extra candidates (from an external note, to verify)
Source: a note from a third party; its dotprod claim is wrong for this phone (no `asimddp`, the dotprod build crashed), its prefill figures (25–50 tok/s on 2B) are optimistic (measured 12–15), and "do not run while charging" is not needed (measured ±3%). Run only after the multi-model eval finishes, so the numbers are not distorted. After each change run `tests/` and record the rate (AGENTS.md rule).
- [x] Pin to the big cores: `-C 0xf0 --cpu-strict 1` with `-t 4`: no gain (pp128 11.05 vs 11.43 baseline, within thermal drift; `logs/bench-tune-qwen35-2b.log`, 2026-10-26).
- [x] Flash attention `-fa on` at pp128: no gain (10.91 vs 11.43). A short context does not show its benefit; still to check at pp1024 (see below).
- [x] KV cache q8_0 with `-fa on` at pp128: no gain (10.76); same caveat, check at pp1024.
- Finding: in a 6-minute bench on Qwen3.5-2B the battery went 39.8 → 46.3 °C (CPU5 45 → 79 °C) and every later variant was slower than the baseline, including a repeated baseline (11.43 → 10.65 tok/s prompt, 6.17 → 5.73 generation), so differences between variants are lost in thermal drift. Any tuning measurement needs a cooled phone and one variant per run, or alternating order (A B A B).
- [ ] Re-test baseline vs `-fa on` + KV q8_0 at pp1024 (the real prompt is ~1200 tokens), alternating order, cooled phone.
- [x] Persist the prompt cache across server restarts (`llama-server --slot-save-path`, `POST /slots/0?action=save|restore`): measured on Qwen3.5-2B (2026-10-26, `logs/bench-slot-qwen35-2b.log`): cold first request 124 s and 128 s (two cold starts), after restoring the saved slot **3 s**; restore itself 75 ms; the slot file is 97 MB for 1329 tokens. Caveats: one run each; the restored query was the same as the saved one (checked afterwards with three different queries after a restore: only 20–25 new prompt tokens were processed each, 4, 6 and 9 s per query, all three tool calls correct); the file is valid only for the same model, flags and a byte-identical system prompt and tool schemas, so it must be rebuilt whenever a tool or the prompt changes (at install, first boot or after a package update); the file size of other models is not measured.
- [ ] Integrate: at service start run `llama-server` with `--slot-save-path`, restore the slot if the file exists and its prompt hash matches, otherwise warm up once and save. (different user queries after a restore already verified.)
- [ ] Quant comparison on Qwen3.5-2B: Q4_K_M vs Q4_0 vs Q8_0 (outcome unclear without dotprod).
- [ ] MTP for Qwen3.5-2B (`--spec-type draft-mtp`, MTP-GGUF) if the current build supports it; low priority, generation is not our bottleneck.
- [ ] Extra candidates to download and eval: Qwen3-1.7B, Qwen3.5-0.8B.
- Not doing: Bonsai 1-bit (accuracy of tool calls matters more than speed); Vulkan on Adreno 630 (no gain expected on small models, driver glitches).
- Measure for each variant: prompt and generation tok/s (`llama-bench`), time of the first request, peak RAM, correct-call rate.

### xLAM-2-1B-fc-r — needs more work (not a fair result yet)
- First eval on the 80-query set (2026-10-26, `~/kuzya/eval-xLAM-2-1B-fc-r-Q4_K_M.txt` on the phone): correct tool 10/80 (12%). 37 of 80 requests failed on the server side: HTTP 500 "The model produced output that does not match the expected peg-native format", i.e. `llama-server --jinja` could not parse the model's tool-call output; another 33 were no call or a wrong tool. So this measures the template/parser fit in this llama.cpp build (`6050b46`), not the model's real ability. Peak RAM 1.24 GB, load 4 s, 6.5 s per query.
- [ ] Inspect the raw model output for a failing query (`llama-cli` or `/completion` with the rendered prompt) and compare with the format the parser expects.
- [ ] Try the chat template from the original xLAM repo (`--chat-template-file`) instead of the one embedded in the GGUF.
- [ ] Try a text-parsing fallback in our own client (parse the model's JSON or tool-call text ourselves, without the server's tool parser), if the template cannot be fixed.
- [ ] Re-run `tests/` and record the rate. Only worth the effort if the other models disappoint: xLAM is a fallback, and its Russian is unverified.

## Этап 2 — инструменты
- [x] 11 инструментов в `internal/tools/` (написаны и покрыты тестами на заглушках; 2026-10-26 проверены на телефоне напрямую через `cmd/kuzya-tool`: все, кроме `wifi_connect` и перезапуска NetworkManager, см. `SUMMARY.md`): `battery_status`, `wifi_status`, `wifi_connect`*, `services_running`, `service_logs`, `service_restart`*, `nosleep_set`*, `volume_get`, `volume_set`*, `mute_set`*, `brightness_set`*. Звёздочка — `Dangerous`.
- [x] Права на телефоне настроены 2026-10-25 (проверено: `sudo -n -l` показывает 7 разрешённых `systemctl restart`, пользователь телефона в `systemd-journal`, журнал читается). Было: `sudo -n` просит пароль, поэтому `service_restart` не сработает без правила sudoers (NOPASSWD для `systemctl restart` из белого списка); `service_logs` читает журнал без sudo, а `<пользователь>` не в группе `systemd-journal`: проверено, системные юниты возвращают `No entries`. Нужно `usermod -aG systemd-journal <пользователь>`. `service_logs` и `service_restart` ограничены одним белым списком (`allowedUnits`). Решает пользователь.
- Всё, что меняет систему, — `Dangerous: true` (подтверждение).
- [x] Параметры (JSON-схема) в полях `Params`/`Required` инструмента; `tools.Schemas()` отдаёт их модели.

## Этап 3 — знания (RAG)
- Собрать `docs/` из `RESTORE.md`, описания конфигов, справочника команд skladenOS.
- Простой локальный поиск (BM25/SQLite FTS5), без эмбеддингов в первой версии.
- Подкладывать топ-N фрагментов в контекст.

## Этап 4 — качество
- [x] `tests/cases.json`: 40 запросов (34 с вызовом, 6 без); раннер `cmd/kuzya-eval`. Qwen3.5-2B: 35/40 (88%), все 34 положительных верны, из 6 негативных верен 1 (`logs/eval-qwen35-2b.txt`). После правки промпта (ответ текстом, если инструмент не подходит): 39/40 (98%), с аргументами 38/40, негативные 6/6 (`logs/eval-qwen35-2b-v2.txt`). Остальные модели не прогонялись.
- Метрика: доля верных вызовов; результат фиксировать в `SUMMARY.md`.
- Набор расширен до 80 запросов (id 41–80 — сленг, опечатки, английский, косвенные и составные фразы, 7 новых негативов); прогон всех 5 моделей идёт на телефоне (`build/run-evals.sh`, заодно время загрузки и пик ОЗУ).
- Result of the 80-query run (2026-10-26, details in `SUMMARY.md`, logs in `logs/eval80/`): Gemma 4 E2B 91% (peak RAM 2.83 GB, median 4.9 s), Qwen3.5-2B 90% (1.73 GB, 6.7 s), Granite 3B 80%, Qwen3.5-4B stopped at 49/80 (battery 50 °C, 3.79 GB, ~16 s per query), xLAM 12% (parser errors). Leaders: Gemma 4 E2B and Qwen3.5-2B. Not measured: Llama 3.2 3B; Qwen3.5-4B incomplete.
- [x] (done 2026-10-26: `max_tokens: 256`, timeout kept at 5 min because the cold first request takes up to 265 s) Add `max_tokens` and a shorter client timeout in `internal/llm` (requests of 155–300 s were seen on Qwen3.5-2B and 4B; likely runaway generation or a re-processed prompt), check the server log for these queries.
- [x] (done 2026-10-26: Gemma 96%, Qwen3.5-2B 94% on 80 queries, `SUMMARY.md`) Rewrite descriptions of `wifi_connect`, `mute_set`, `nosleep_set` (on/off meaning, examples of phrasing), re-run `tests/` on the two leaders.
- [ ] Thermal rule for long phone runs: watch `thermal_zone*` and `qcom-battery`, stop at 50 °C battery; CPU reached 95 °C under load, which may distort timings (the unplugged `llama-bench` runs had 39–41 °C).
- [x] Ещё модели на 80 запросах прогнаны (2026-10-26): Qwen3.5-0.8B 76%, Qwen3-1.7B 82%, Llama 3.2 3B остановлена на 68/80 (батарея 51,8 °C, примерно 82%); лидеры по-прежнему Gemma 4 E2B (96%) и Qwen3.5-2B (94%), см. `SUMMARY.md`. LFM2.5 и SmolLM3 — только если лидеры разочаруют.
- [x] (done 2026-10-26: 133/148, 90%, батарея −14% за 16,5 мин, см. `SUMMARY.md`) Перепрогнать все 148 запросов (`tests/cases-all.json`) на Gemma 4 E2B с вариантом D.
- [x] Сложный набор `tests/cases-hard2.json` (id 81–120, 40 запросов, 18 негативов): Gemma 4 E2B прогнана — 36/40 (90%), промахи: «размуть», «Keenetic-9135 включи пожалуйста», «перезапусти все службы» и «останови NetworkManager» дали `service_restart`; Qwen3.5-2B прогнана — 28/40 (70%), негативы 10/18; Gemma 4 E2B заметно лучше. Промпт v3 («что умеет», «остановить ≠ перезапустить») прогнан на Gemma: 130/136 (96%), сложный набор 37/40, отложенный 16/16 против 15/16 у старого промпта — прирост в один запрос на каждый набор, т.е. в пределах шума; нужны ещё отложенные запросы для надёжного вывода (`CASES=cases-hard2.json ./run-evals.sh -hard2 <модели>`), по одной, с остыванием до 35 °C.
- Сравнить модели из короткого списка на одном наборе; победителя оставить.
- Если < ~80% — улучшать промпт/описания; затем LoRA на нескольких сотнях примеров.

## Stage 4b — LoRA fine-tuning (decided 2026-10-26)
Why: prompt-only tool selection is brittle on small models. Adding one tool (`brightness_get`) or changing a description made unrelated `wifi_status` phrasings fail (ids 7, 45, 46; 86/90 → 82/90 on the same queries, repeatable), and every description tweak has to be re-verified on the whole set. A model fine-tuned on our 11 tools should be stable, allow much shorter tool descriptions (the prompt is about 1300 tokens now, the cold first request costs 100–265 s and about 2% of the battery) and may let a smaller, cheaper model (Qwen3.5-0.8B scored 76% without tuning, half the RAM of Gemma) reach the accuracy of Gemma 4 E2B (91–96% on our sets).
- Hardware for training: Apple M2 Pro, 16 GB, 84 GB free, Python `mlx` present, `mlx_lm` to be installed. Conversion back needs the scripts from the `llama.cpp` repository (merge the adapter, `convert_hf_to_gguf.py`, `llama-quantize` to Q4_K_M); our Docker build only has the binaries.
- [ ] Baseline without tuning is recorded (Gemma 4 E2B, Qwen3.5-2B, Qwen3.5-0.8B on `tests/cases-all.json`), after the relative-request experiment (variant D with `*_step` tools, see `SUMMARY.md`; `delta` was dropped).
- [ ] Set `temperature` to 0 for tool calling (now 0.2) and re-run the baseline: part of the run-to-run jumps may be sampling noise; makes all comparisons fairer.
- [ ] Training data generator: templates plus paraphrases in Russian (slang, typos, English mixed in, indirect and compound phrasing, numbers in words, relative requests "ярче/потише" (`*_step` tools with `direction`), negatives that are close to real tools), several hundred to a few thousand examples; separate `train`, `dev` and `test` files. **The training phrasings must not overlap with `tests/cases*.json`** (otherwise the rate proves nothing); keep a small hand-written held-out set like `tests/cases-held.json`. Negatives must be a large share, otherwise the model forgets to refuse in text.
- Terms status (2026-10-26): both DeepSeek documents pasted by the owner were read in full: **Open Platform Terms of Service** (release 2026-04-22, effective 2026-04-29) and the general **Terms of Use** (last update 2026-03-27). Both say in §4.2(3) that Inputs and Outputs may be used for "training other models (such as model distillation)" if the use is legal and follows the Terms; §4.2(2) assigns Output rights to the user; §4.1 the user answers for Inputs (rights, personal data). The Open Platform Terms are the Specific Terms for the API and prevail over the general ones. Duties and limits that matter here: general ToU §3.1: when **publishing or distributing Outputs** (e.g. a dataset in a public repository) verify their accuracy, **state that they are AI-generated** and publish nothing that breaks the Terms; §3.5(3): no reverse engineering of the models and no copying of the Services' content with **robots, spiders or other automatic setups**, so bulk generation goes through the **API only, never by scripting the web chat**; §3.6(4) no re-selling or sub-licensing the Services; §3.4 content restrictions (not an issue for our phrasings); §4.3 DeepSeek may use de-identified Inputs and Outputs to improve its services (the opt-out toggle "Improve the model for everyone" is in the app, how it works for the API is not stated); Open Platform §2.2 the API key must not be shared or exposed; §5.2–5.3 / ToU §6.2 no use of the DeepSeek name or logo that implies partnership; Open Platform §8.1 disclose AI-generated output to end users of an application; §10 PRC law and Hangzhou courts; §7.2 the account can be restricted without notice; §11.1 terms change (Open Platform: 7 days), so record the revision used. Not covered by the texts read: where data is stored and the retention of API inputs (DeepSeek Privacy Policy, **not read**). Practice if the API is used: key only in an environment variable, never in git; send only the owner's seed phrases, never data from the phone; keep a provenance note next to every data file (source, date, terms revisions above, "AI-generated" label); mention DeepSeek only as a factual source note, not as a partner.
- [ ] **Data source and terms (check before generating anything):** training text must come from a source whose terms allow training another model on its outputs. Candidates: (1) phrasings written by the owner (seeds) plus (2) paraphrases from a **local open-weights model** run on the Mac (Qwen under Apache 2.0, DeepSeek weights under MIT: no service terms, nothing leaves the machine); a hosted API (Anthropic, DeepSeek, others) only after reading its current terms on using outputs to train models, and never with private data from the phone. Text written by Claude is not used as training data unless the owner has confirmed that the applicable Anthropic terms allow it (the generator code and the labelling scheme are ordinary software and not affected). Record the source and licence of every data file next to it.
- [ ] Check that `mlx_lm` can fine-tune the candidate architectures (Qwen3.5 and Gemma 4 are new, Qwen3-1.7B is the safe fallback); trial run on the smallest suitable model to verify the whole chain: LoRA → merged weights → GGUF Q4_K_M → `llama-server --jinja` on the phone → `tests/`.
- [ ] Shorten tool descriptions after tuning and re-measure the prompt size, the cold first request and the correct-call rate.
- [ ] Pick the model for the full training by the measured rate, speed, peak RAM and battery cost per query (about 0.1% per query for Gemma 4 E2B now); update `SUMMARY.md`.
- Decision rule: tuned model wins only if it beats the prompt-only baseline on the held-out set and on negatives, not just on the training-like phrasings.

## Этап 5 — интеграция
- Автозапуск `llama-server` по требованию (не держать в ОЗУ постоянно).
- Сквозная проверка через агента на телефоне сделана (2026-10-26, Gemma 4 E2B): подтверждение, отказ («Отменено: …»), пустой ответ модели после инструмента заменяется результатом инструмента; `service_restart NetworkManager` проверен (работает, ~5 с, связь Wi-Fi и USB возвращается); `wifi_connect` из SSH не работает (polkit: «Not authorized to control networking»), но из локальной сессии SXMO (через `swaymsg exec`) работает: подключение за 9 с, поэтому Кузя запускается из пункта меню SXMO (`deploy/sxmo/kuzya`), правило polkit не нужно.
- Пункт меню SXMO и/или голосовой/текстовый ввод.
- Упаковка в deb и включение в образ skladenOS.

## Этап 6 — безопасность
- Журнал всех вызовов инструментов.
- Лимит шагов, таймауты, отказ по умолчанию для опасных действий.
