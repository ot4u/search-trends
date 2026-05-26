# Мониторинг

Для локальной разработки и демо хватает **Prometheus** (метрики и правила алертов), **Grafana** (дашборд), **структурированных логов** и **pprof**. Без Kubernetes, OpenTelemetry, Loki и Jaeger — намеренно, чтобы стек поднимался одной командой.

---

## Запуск

```bash
make docker-up
```

- Prometheus — http://localhost:9090 (алерты: вкладка Alerts)
- Grafana — http://localhost:3000, логин `admin` / `admin`
- метрики приложения — http://localhost:8080/metrics

Конфиги лежат в `prometheus/`:

- `prometheus.yml` — откуда собирать метрики с `app:8080`
- `alerts.yml` — правила
- `grafana.json` — дашборд
- `grafana/provisioning/` — автоподключение источника данных и дашборда

Файл `grafana/provisioning/datasources/prometheus.yml` — это **настройка Grafana** (куда ходить за графиками), а не конфиг сервера Prometheus.

---

## Зачем так

- **Prometheus** — снятие метрик, PromQL, проверка порогов
- **Grafana** — RPS, задержки, очередь, лаг NATS на одном экране
- **slog** — в логах request id, статус, задержка, размер очереди
- **pprof** — разбор CPU, памяти и горутин при отладке

Отдельный Alertmanager в compose не поднят: для прода правила из `alerts.yml` обычно переносят в уже существующий Alertmanager.

---

## Метрики HTTP

- `search_trends_http_requests_total` — сколько запросов, с разбивкой по методу, пути и коду ответа
- `search_trends_http_request_duration_seconds` — гистограмма задержек (p50, p95, p99 в Grafana)

---

## Метрики приёма событий

- `search_trends_events_total{outcome}` — принято, стоп-лист, переполнение, устарело и т.д.
- `search_trends_events_dropped_total` — отброшено по capacity / stale / future
- `search_trends_antifraud_downweighted_total` — принято с пониженным весом
- `search_trends_applied_weight_total` — суммарный вес принятых
- `search_trends_ingest_queue_size` — сколько событий в очереди сейчас
- `search_trends_ingest_queue_capacity` — размер очереди (`QUEUE_SIZE`)

---

## Состояние в памяти

- `search_trends_unique_queries` — сколько уникальных запросов в окне
- `search_trends_memory_alloc_bytes` — текущая куча (обновление раз в 5 с)

---

## NATS

- `search_trends_nats_messages_total{outcome}` — поставлено в очередь, битый JSON, ошибка enqueue
- `search_trends_nats_consumer_lag_seconds` — оценка отставания по времени события

---

## Снимки Top-N

- `search_trends_snapshot_age_seconds` — сколько секунд с последнего обновления
- `search_trends_snapshot_refresh_duration_seconds` — сколько длился пересчёт
- `search_trends_snapshot_refresh_total` — счётчик обновлений

---

## Grafana

Дашборд `prometheus/grafana.json`: RPS и задержки HTTP, поток событий, очередь, уникальные запросы, память, лаг consumer, возраст и длительность refresh снимка.

---

## Алерты (`prometheus/alerts.yml`)

- **IngestQueueHighUsage** — очередь заполнена больше чем на 90%, минута — писатель не успевает
- **SnapshotStale** — снимку больше 5 секунд — сломался тик или refresh
- **HTTPP99LatencyHigh** — p99 HTTP больше 200 мс, две минуты
- **NATSConsumerLagHigh** — лаг больше 10 с

---

## Логи

На каждый HTTP-запрос пишется: `request_id`, метод, путь, код, задержка, `queue_size`. Удобно связать всплеск p99 с заполнением очереди.

---

## pprof

```bash
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

- **profile** — если горит CPU (пересчёт Top-N, refresh)
- **heap** — рост уникальных запросов, риск OOM
- **goroutine** — утечки писателя, consumer, HTTP

В проде endpoint лучше закрыть с сети; в репозитории он включён для локальной отладки.

---

## На что смотреть при проблемах

- очередь к 90% и выше — ingest быстрее писателя
- возраст снимка > 5 с — завис тик или паника в писателе
- p99 HTTP растёт при стабильном RPS — GC, перегруз, большие ответы
- лаг consumer растёт — NATS быстрее обработки
- `unique_queries` у потолка — срабатывает лимит кардинальности
- растёт `antifraud_downweighted` — частые повторы одного запроса в секунду

---

## CI

В GitHub Actions гоняются тесты, race detector и линтер. Метрики и дашборды в CI не проверяются.

---

## См. также

- [architecture.md](architecture.md) — откуда берутся очередь, снимки, исходы событий
- [load_testing.md](load_testing.md) — нагрузка и сверка с метриками
- [benchmarks.md](benchmarks.md) — скорость операций в коде
