# Руководство по использованию EasyProfiler MCP Server

## Быстрый старт

### 1. Сборка проекта

```bash
cd easyprofiler-mcp
go mod download
go build -o easyprofiler-mcp
```

### 2. Настройка MCP клиента

Добавьте сервер в конфигурацию вашего MCP клиента (например, Claude Desktop).

Для Windows (`%APPDATA%\Claude\claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "easyprofiler": {
      "command": "C:\\path\\to\\easyprofiler-mcp\\easyprofiler-mcp.exe"
    }
  }
}
```

Для macOS/Linux (`~/.config/claude/claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "easyprofiler": {
      "command": "/path/to/easyprofiler-mcp/easyprofiler-mcp"
    }
  }
}
```

### 3. Использование в Claude Desktop

После перезапуска Claude Desktop вы сможете использовать инструменты анализа:

## Примеры использования

### Загрузка профиля

```
Загрузи профиль из файла C:\profiles\myapp.prof
```

Сервер вернет информацию о профиле:
```json
{
  "status": "success",
  "file": "C:\\profiles\\myapp.prof",
  "version": "0x2010000",
  "pid": 12345,
  "total_duration": "5.234s",
  "threads_count": 8,
  "blocks_count": 15432,
  "descriptors_count": 245,
  "bookmarks_count": 0
}
```

### Анализ проблем производительности

```
Проанализируй проблемы производительности в загруженном профиле
```

Результат:
```json
{
  "total_issues": 12,
  "summary": "Found 12 performance issues (3 high, 6 medium, 3 low)",
  "by_severity": {
    "high": [
      {
        "type": "Hot Function",
        "description": "Function 'ProcessData' consumes 35.2% of total time (1.843s total, 1523 calls, avg 1.21ms)",
        "location": "ProcessData (src/processor.cpp:145)"
      },
      ...
    ],
    "medium": [
      {
        "type": "Long Blocking Operation",
        "description": "Block 'DatabaseQuery' took 234ms",
        "location": "src/database.cpp:89",
        "duration": "234ms",
        "thread_name": "Worker-3"
      },
      ...
    ],
    "low": [...]
  }
}
```

### Топ самых медленных блоков

```
Покажи 20 самых медленных блоков
```

Результат:
```json
[
  {
    "rank": 1,
    "name": "DatabaseQuery",
    "file": "src/database.cpp",
    "line": 89,
    "duration": "234ms",
    "duration_ns": 234000000,
    "thread_id": 8192,
    "thread_name": "Worker-3"
  },
  ...
]
```

### Статистика по потокам

```
Покажи статистику использования времени по потокам
```

Результат:
```json
[
  {
    "thread_id": 4096,
    "thread_name": "Main Thread",
    "total_duration": "2.145s",
    "block_count": 3421,
    "context_switches": 234,
    "avg_block_duration": "627µs",
    "percent_of_total": "41.02%"
  },
  {
    "thread_id": 8192,
    "thread_name": "Worker-1",
    "total_duration": "1.823s",
    "block_count": 2876,
    "context_switches": 567,
    "avg_block_duration": "634µs",
    "percent_of_total": "34.84%"
  },
  ...
]
```

### Горячие точки (hotspots)

```
Найди функции с наибольшим суммарным временем выполнения
```

Результат:
```json
[
  {
    "rank": 1,
    "name": "ProcessData",
    "file": "src/processor.cpp",
    "line": 145,
    "total_duration": "1.843s",
    "call_count": 1523,
    "avg_duration": "1.21ms",
    "percent_of_total": "35.22%"
  },
  {
    "rank": 2,
    "name": "RenderFrame",
    "file": "src/renderer.cpp",
    "line": 67,
    "total_duration": "892ms",
    "call_count": 60,
    "avg_duration": "14.87ms",
    "percent_of_total": "17.05%"
  },
  ...
]
```

## Типы выявляемых проблем

### 1. Long Blocking Operation
Операции блокировки длительностью более 100ms.

**Severity:**
- High: > 500ms
- Medium: 100-500ms

**Решение:** Оптимизировать или распараллелить операцию.

### 2. Hot Function
Функции, занимающие более 10% общего времени выполнения.

**Severity:**
- High: > 30%
- Medium: 20-30%
- Low: 10-20%

**Решение:** Оптимизировать алгоритм, кэшировать результаты, уменьшить количество вызовов.

### 3. Thread Imbalance
Дисбаланс нагрузки между потоками (разница более чем в 2 раза).

**Severity:** Medium

**Решение:** Перераспределить работу между потоками, использовать thread pool.

### 4. Excessive Context Switches
Чрезмерное количество переключений контекста (> 1000).

**Severity:** Medium

**Решение:** Уменьшить количество потоков, использовать батчинг операций.

## Workflow анализа производительности

1. **Загрузите профиль**
   ```
   load_profile(file_path="path/to/profile.prof")
   ```

2. **Получите общий анализ проблем**
   ```
   analyze_performance_issues()
   ```

3. **Детализируйте проблемные области:**
   - Для горячих функций: `get_hotspots(limit=20)`
   - Для долгих операций: `get_slowest_blocks(limit=20)`
   - Для проблем с потоками: `get_thread_statistics()`

4. **Оптимизируйте код** на основе полученных данных

5. **Повторите профилирование** и сравните результаты

## Советы по оптимизации

### Если найдены Hot Functions:
- Проверьте алгоритмическую сложность (O(n²) → O(n log n))
- Используйте профилирование на уровне строк кода
- Рассмотрите кэширование результатов
- Проверьте, можно ли уменьшить количество вызовов

### Если найдены Long Blocking Operations:
- Переместите операции ввода-вывода в фоновые потоки
- Используйте асинхронные операции
- Разбейте большие операции на более мелкие части
- Рассмотрите использование буферизации

### Если найден Thread Imbalance:
- Используйте динамическое распределение задач
- Рассмотрите thread pool с work stealing
- Проверьте размер задач - они должны быть примерно одинаковыми
- Измерьте накладные расходы на создание потоков

### Если найдены Excessive Context Switches:
- Уменьшите количество потоков до количества ядер CPU
- Используйте батчинг операций
- Избегайте частой блокировки/разблокировки мьютексов
- Рассмотрите lock-free структуры данных

## Ограничения

1. **Размер файла:** Парсер загружает весь профиль в память. Для очень больших файлов (>1GB) может потребоваться много RAM.

2. **Точность анализа:** Анализатор использует эвристики для определения проблем. Не все выявленные проблемы требуют немедленного решения.

3. **Вложенные блоки:** При анализе горячих точек учитывается полное время блока, включая дочерние блоки.

## Troubleshooting

### "No profile loaded. Use load_profile first."
Сначала загрузите профиль с помощью `load_profile`.

### "Failed to parse profile: invalid file signature"
Файл не является валидным .prof файлом EasyProfiler.

### "Failed to parse profile: unsupported version"
Версия формата файла слишком старая (< 0.1.0).

### Сервер не отвечает
Проверьте:
1. Правильность пути к исполняемому файлу в конфигурации
2. Права на выполнение файла
3. Логи Claude Desktop для деталей ошибки
