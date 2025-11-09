# Инструкция по установке и настройке

## Быстрая установка

### Шаг 1: Сборка проекта

```bash
cd easyprofiler-mcp
go mod download
go build -o easyprofiler-mcp.exe  # Windows
# или
go build -o easyprofiler-mcp      # Linux/macOS
```

### Шаг 2: Настройка Claude Desktop

#### Windows

1. Откройте конфигурационный файл Claude Desktop:
   ```
   %APPDATA%\Claude\claude_desktop_config.json
   ```

2. Добавьте следующую конфигурацию:
   ```json
   {
     "mcpServers": {
       "easyprofiler": {
         "command": "C:\\path\\to\\easyprofiler-mcp\\easyprofiler-mcp.exe"
       }
     }
   }
   ```

   Замените `C:\\path\\to\\easyprofiler-mcp\\` на реальный путь к директории проекта.

#### macOS

1. Откройте конфигурационный файл:
   ```bash
   nano ~/.config/claude/claude_desktop_config.json
   ```

2. Добавьте:
   ```json
   {
     "mcpServers": {
       "easyprofiler": {
         "command": "/path/to/easyprofiler-mcp/easyprofiler-mcp"
       }
     }
   }
   ```

#### Linux

1. Откройте конфигурационный файл:
   ```bash
   nano ~/.config/claude/claude_desktop_config.json
   ```

2. Добавьте:
   ```json
   {
     "mcpServers": {
       "easyprofiler": {
         "command": "/path/to/easyprofiler-mcp/easyprofiler-mcp"
       }
     }
   }
   ```

### Шаг 3: Перезапустите Claude Desktop

После добавления конфигурации полностью закройте и перезапустите Claude Desktop.

## Проверка установки

1. Откройте Claude Desktop
2. Введите: "Какие инструменты доступны для анализа профилирования?"
3. Claude должен показать доступные инструменты EasyProfiler MCP сервера

## Пример использования

### 1. Загрузите профиль

```
Загрузи профиль из файла C:\profiles\myapp.prof
```

или для Linux/macOS:
```
Загрузи профиль из файла /home/user/profiles/myapp.prof
```

### 2. Анализ проблем

```
Проанализируй проблемы производительности
```

### 3. Детальный анализ

```
Покажи топ 20 самых медленных функций
```

```
Покажи статистику по потокам
```

```
Найди горячие точки в коде
```

## Требования

- Go 1.21 или выше
- Claude Desktop
- Файлы профилирования EasyProfiler (.prof)

## Структура проекта

```
easyprofiler-mcp/
├── analyzer/           # Модуль анализа производительности
│   └── analyzer.go
├── parser/            # Парсер .prof файлов
│   ├── reader.go
│   └── types.go
├── main.go            # Главный файл сервера
├── go.mod             # Go модуль
├── README.md          # Основная документация
├── USAGE.md           # Руководство по использованию
├── FORMAT.md          # Спецификация формата .prof
├── INSTALL.md         # Инструкция по установке (этот файл)
├── Makefile           # Makefile для сборки
└── LICENSE            # Лицензия MIT
```

## Сборка для разных платформ

### Windows (из Linux/macOS)
```bash
GOOS=windows GOARCH=amd64 go build -o easyprofiler-mcp.exe
```

### Linux (из Windows/macOS)
```bash
GOOS=linux GOARCH=amd64 go build -o easyprofiler-mcp
```

### macOS (из Windows/Linux)
```bash
GOOS=darwin GOARCH=amd64 go build -o easyprofiler-mcp
```

### ARM64 (для Apple Silicon)
```bash
GOOS=darwin GOARCH=arm64 go build -o easyprofiler-mcp
```

## Использование через Makefile

Проект включает Makefile для удобства:

```bash
# Сборка для текущей платформы
make build

# Сборка для всех платформ
make build-all

# Очистка артефактов сборки
make clean

# Скачивание зависимостей
make deps

# Запуск тестов
make test

# Форматирование кода
make fmt
```

## Troubleshooting

### Ошибка: "command not found"

Убедитесь, что:
1. Путь к исполняемому файлу указан правильно
2. Файл имеет права на выполнение (Linux/macOS):
   ```bash
   chmod +x easyprofiler-mcp
   ```

### Ошибка: "Failed to open file"

Проверьте:
1. Путь к .prof файлу корректен
2. Файл существует и доступен для чтения
3. На Windows используйте двойные обратные слэши в пути: `C:\\path\\to\\file.prof`

### Сервер не отвечает

1. Проверьте логи Claude Desktop
2. Убедитесь, что JSON конфигурация валидна
3. Попробуйте запустить сервер вручную для проверки:
   ```bash
   ./easyprofiler-mcp.exe  # Windows
   ./easyprofiler-mcp      # Linux/macOS
   ```

### Ошибка при сборке

Если возникают ошибки при сборке:
```bash
go clean -modcache
go mod tidy
go mod download
go build
```

## Обновление

Для обновления на новую версию:

1. Получите обновленный код
2. Пересоберите проект:
   ```bash
   make clean
   make build
   ```
3. Перезапустите Claude Desktop

## Удаление

1. Удалите запись из `claude_desktop_config.json`
2. Удалите директорию проекта
3. Перезапустите Claude Desktop

## Поддержка

Для сообщений об ошибках и предложений по улучшению создайте issue в репозитории проекта.

## Разработка

### Запуск в режиме разработки

```bash
go run main.go
```

### Тестирование парсера

Создайте тестовый файл `parser/reader_test.go`:

```go
package parser

import "testing"

func TestReadProfile(t *testing.T) {
    reader, err := NewReader("../testdata/sample.prof")
    if err != nil {
        t.Fatal(err)
    }
    defer reader.Close()

    profile, err := reader.Parse()
    if err != nil {
        t.Fatal(err)
    }

    if profile.Header.Signature != EasyProfilerSignature {
        t.Errorf("Invalid signature")
    }
}
```

Запуск тестов:
```bash
go test ./...
```

## Лицензия

MIT License - см. файл LICENSE для деталей.
