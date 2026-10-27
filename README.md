<p align="center"><img src="assets/kuzya-mid-glasses.svg" width="128" alt="Kuzya"></p>

# Кузя

Модели: [nafanya-1.7b](https://huggingface.co/MityaLabs/nafanya-1.7b) · [nafanya-0.8b](https://huggingface.co/MityaLabs/nafanya-0.8b)

Локальный офлайн-агент-домовой для skladenOS и Skladen (Poco F1).
Настраивает систему и помогает с ней работать без сети: маленькая квантованная
модель (llama.cpp) + белый список инструментов + поиск по локальной документации.

```
kuzya "подключи Wi-Fi"
```

## Принципы
- Модель не обучается с нуля: берётся готовая открытая (Qwen/Gemma/Llama/Phi, 1.5–3B, Q4).
- Никакого произвольного shell: только инструменты из `internal/tools/`.
- Опасные действия требуют подтверждения пользователя.
- Знания о системе — в `docs/`, подтягиваются в контекст (RAG).
- Дообучение (LoRA) сделано: модель **nafanya-1.7b** (Qwen3-1.7B + LoRA), воркспейс
  `../nafanya`; компактный промпт и короткие описания инструментов — основной режим,
  подробные остались в `agent.SystemFull` / `tools.SchemasFull()` для базовых моделей.

## Структура
Агент написан на Go (статический бинарник, кросс-сборка с Мака).
- `cmd/kuzya/` — точка входа: `kuzya` (интерактивный режим) или `kuzya "запрос"` (один запрос); префикс `?` — ответить текстом без инструментов, «кто ты» и «что умеешь» отвечают статически
- `internal/agent/` — цикл «модель → инструмент → результат → модель», обрезка истории
- `internal/tools/` — реестр инструментов (с флагом `Dangerous`)
- `internal/llm/` — клиент к `llama-server` (HTTP, OpenAI-совместимый API, адрес в `KUZYA_URL`)
- `build/` — Dockerfile сборки `llama.cpp` под телефон
- `docs/` — документация для поиска
- `tests/` — набор тестовых запросов для замера доли верных вызовов
- `logs/` — логи прогонов

Сборка под телефон: `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags='-s -w' -o build/out/kuzya ./cmd/kuzya`

## Build and run / Сборка и запуск
- `llama.cpp` for the phone (aarch64, glibc ≤ 2.43), built in Docker on the Mac / собирается в Docker на Маке:
  `docker build --platform linux/arm64 --target out --output type=local,dest=build/out -f build/Dockerfile build`
- Copy `build/out/*` and a `.gguf` model to the phone (`~/kuzya/bin`, `~/kuzya/models`); models are not in git / модели не в git.
- The interactive REPL clears the screen on start (disable with `KUZYA_NOCLEAR=1`). / Интерактивный режим очищает экран при старте (`KUZYA_NOCLEAR=1` отключает).
- Run `llama-server --jinja -m <model>.gguf` on the phone, then `kuzya`. Server address: `KUZYA_URL`, default `http://127.0.0.1:8080/v1/chat/completions`.
- Talk vs tools / Разговор и инструменты: `? вопрос` — разговор; если заданы `KUZYA_REMOTE_URL`, `KUZYA_REMOTE_KEY`, `KUZYA_REMOTE_MODEL`, он уходит на удалённую OpenAI-совместимую модель, иначе отвечает локальная (без инструментов). `@ вопрос` или `! вопрос` — удалённая модель получает те же схемы инструментов и может попросить телефон выполнить действие (`!` удобно в русской раскладке); опасные действия всё равно подтверждает пользователь, результаты инструментов уходят на сервер. Это единственный сетевой выход и только по явному тегу. Если 0.8B в режиме `?` отвечает «не умею», агент сам перезапускает сервер на 1.7B и повторяет запрос; обе локальные модели на бытовые вопросы отвечают отказом, поэтому для настоящего разговора нужна удалённая.
- Remote settings / Настройка удалённой модели: `/remote setup` спрашивает URL, ключ (ввод скрыт) и модель и сохраняет в `~/kuzya/remote.env` с правами 600; `/remote show` показывает конфиг с замаскированным ключом. Файл главнее переменных окружения. / `/remote setup` stores URL, key (hidden input) and model in `~/kuzya/remote.env` (mode 600).
- Remote-only / Только удалённая модель: `/remote` в REPL переключает режим — все запросы идут удалённой модели, локальный сервер останавливается (память и батарея свободны), инструменты выполняются на телефоне с подтверждением. `/remote off` возвращает локальную модель. / `/remote` toggles remote-only: the local server stops, all requests go to the remote model, tools still run locally.
- Model switch / Переключение модели: `KUZYA_MODEL=nafanya-1.7b-Q4_K_M kuzya "…"` (default) or `KUZYA_MODEL=nafanya-0.8b-Q4_K_M kuzya "…"`. `kuzya-server` restarts itself when the model changes and keeps a separate slot cache per model. / При смене модели сервер перезапускается сам, кэш слота у каждой модели свой.

| Model | Fresh sets (rules) | Peak RAM | Warm energy | Use |
|---|---:|---:|---:|---|
| nafanya-1.7b | 98% | 1.71 GB | 8.9 J/query | default, accuracy first |
| nafanya-0.8b | 88% | 0.92 GB | 4.4 J/query | memory first |
- Tests / тесты: `go test ./...`
