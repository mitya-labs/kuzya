# Обзор малых моделей для Кузи (на 2026-10-25)

Источники: блоги и агрегаторы, цифры расходятся, считать предварительными.
Решение принимать по замерам на телефоне (см. PLAN.md, этап 1–2).

Ограничение: Poco F1, ОЗУ 5,5 ГБ всего, свободно ~4,6 ГБ → файл модели в Q4 желательно ≤ ~3 ГБ.

| Модель | Что известно | Оценка для нас |
|---|---|---|
| Qwen3.5 0.8B / 2B / 4B (2026-03-02) | 4B в Q4 ~3 ГБ, 2B и 0.8B <2 ГБ; вызов инструментов; контекст 262K; режимы с рассуждением и без; Apache 2.0 | Главный кандидат (2B и 4B). Замеров BFCL для малых версий не нашёл |
| Gemma 4 E2B / E4B | Нативный вызов инструментов, Apache 2.0, нужна свежая сборка llama.cpp и `--jinja`; E4B по размеру загрузки (6,6–9,6 ГБ) тяжёлая; E2B заметно слабее E4B | Проверить E2B; E4B вряд ли влезет |
| xLAM-2-1b-fc-r (Salesforce) | Специализация на вызове функций; BFCL v4 ~30,44 против ~10,82 у Llama-3.2-1B (по одному блогу) | Самый компактный специалист; русский под вопросом |
| MiniCPM5-1B | Модель 1B для устройств; в SGLang есть парсер вызовов | Дополнительный, данных мало |
| Granite 4.1 3B (IBM) | BFCL v3 ~60,8 (трекер), BFCL v4 — 16 место | Сильный по инструментам; проверить русский |
| LFM2.5-2.6B (Liquid AI) | BFCL v4 — 15 место | Проверить; скорость на ARM заявлена хорошей |
| Nemotron Nano 4B (NVIDIA) | RL-обучена под вызов инструментов | Дополнительный |
| Llama 3.2 3B | BFCL v2: 3B ~67%, 1B ~25,7% | Контрольная модель; русский официально не поддерживается |
| Phi-4-mini 3.8B | ~3 ГБ в Q4; по инструментам слабее | Скорее нет |
| SmolLM3 3B | Вызов инструментов есть, BFCL не найден | Дополнительный |

## Короткий список для замеров
1. Qwen3.5-4B Q4
2. Qwen3.5-2B Q4
3. Gemma 4 E2B Q4
4. Granite 4.1 3B Q4
5. Llama 3.2 3B Q4 (контроль)
+ xLAM-2-1b-fc-r, если нужна минимальная скорость/ОЗУ.

## Метрики
- доля верных вызовов инструментов на 30–50 запросах (в основном русских);
- токены/с (prompt и generation);
- пиковая ОЗУ, время загрузки;
- нагрев/батарея при длительной работе.

## Оговорки
- Новые архитектуры требуют свежего llama.cpp — собирать из исходников.
- Официальные результаты BFCL смотреть на gorilla.cs.berkeley.edu, не по блогам.

## Ссылки
- https://artificialanalysis.ai/articles/qwen3-5-small-models
- https://unsloth.ai/docs/models/qwen3.5
- https://x.com/Alibaba_Qwen/status/2028460046510965160
- https://www.mindstudio.ai/blog/gemma-4-edge-deployment-e2b-e4b-models
- https://huggingface.co/unsloth/gemma-4-E2B-it-GGUF
- https://avenchat.com/blog/run-gemma-4-with-llama-cpp
- https://github.com/ShishirPatil/gorilla/blob/main/berkeley-function-call-leaderboard/CHANGELOG.md
- https://llm-stats.com/benchmarks/bfcl-v4
- https://www.llmreference.com/benchmark/bfcl-v3
- https://medium.com/@minhle_0210/5-tiny-language-models-for-tool-calling-part-3-ebcda32c2518
- https://www.digitalapplied.com/blog/small-language-models-on-device-agents-2026-guide
