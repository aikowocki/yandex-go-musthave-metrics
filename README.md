# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Профилирование и оптимизация памяти (iter17)

### Методика

1. Подключён `net/http/pprof` на отдельном порту (`:6060` сервер, `:6061` агент)
2. Нагрузка через `hey` (3×10000 запросов: `/updates`, `/update`, `/`)
3. Профиль: `alloc_space` (cumulative allocations)

### Результат оптимизации сервера

```
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof

File: server
Type: alloc_space
Showing nodes accounting for -10592.07MB, 95.97% of 11037.23MB total

      flat  flat%   sum%        cum   cum%
-8619.52MB 78.09% 78.09% -10546.91MB 95.56%  compress/flate.NewWriter
-1872.85MB 16.97% 95.06%  -1872.85MB 16.97%  compress/flate.(*compressor).initDeflate
  -82.18MB  0.74% 95.81%    -82.18MB  0.74%  compress/flate.(*huffmanEncoder).generate
  -29.02MB  0.26% 96.07%    -54.54MB  0.49%  compress/flate.newHuffmanBitWriter
```

**Снижение: -10,592 MB (96%)**

### Результат оптимизации агента

```
go tool pprof -top -diff_base=profiles/agent_base.pprof profiles/agent_result.pprof

File: agent
Type: alloc_space
Showing nodes accounting for -27734.09kB, 79.70% of 34799.20kB total

         flat  flat%   sum%        cum   cum%
-18954.31kB 54.47% 54.47% -20047.82kB 57.61%  compress/flate.NewWriter
 -1093.51kB  3.14% 62.12%  -1093.51kB  3.14%  compress/flate.(*compressor).initDeflate
 -1025.12kB  2.95% 64.97%  -2563.22kB  7.37%  gopsutil/cpu.TimesWithContext
```

**Снижение: -27,734 KB (80%)**

### Что было оптимизировано

| Оптимизация | Файл | Эффект |
|-------------|------|--------|
| `sync.Pool` для `gzip.Writer` (сервер) | `middleware/gzip.go` | -10.5 GB allocs |
| `sync.Pool` для `gzip.Reader` (сервер) | `middleware/gzip.go` | -51 KB/req |
| `sync.Pool` для `gzip.Writer` (агент) | `agent/sender.go` | -19 KB allocs |
| Убрать `map[string]any` boxing | `agent/collector.go` | 0 allocs в CollectMetrics |
| Pre-allocate слайса | `agent/sender.go` | -39% allocs в CollectBatch |
