<p align="center"><img src="assets/kuzya-mid-glasses.svg" width="128" alt="Kuzya"></p>

# Кузя

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
- `cmd/kuzya/` — точка входа: `kuzya` (интерактивный режим) или `kuzya "запрос"` (один запрос)
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
- Run `llama-server --jinja -m <model>.gguf` on the phone, then `kuzya`. Server address: `KUZYA_URL`, default `http://127.0.0.1:8080/v1/chat/completions`.
- Model switch / Переключение модели: `KUZYA_MODEL=nafanya-1.7b-Q4_K_M kuzya "…"` (default) or `KUZYA_MODEL=nafanya-0.8b-Q4_K_M kuzya "…"`. `kuzya-server` restarts itself when the model changes and keeps a separate slot cache per model. / При смене модели сервер перезапускается сам, кэш слота у каждой модели свой.

| Model | Fresh sets (rules) | Peak RAM | Warm energy | Use |
|---|---:|---:|---:|---|
| nafanya-1.7b | 98% | 1.71 GB | 8.9 J/query | default, accuracy first |
| nafanya-0.8b | 88% | 0.92 GB | 4.4 J/query | memory first |
- Tests / тесты: `go test ./...`
