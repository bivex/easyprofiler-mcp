# EasyProfiler .prof File Format Specification

Полная спецификация бинарного формата файлов EasyProfiler на основе анализа исходного кода.

## Общая структура файла

```
[Header]
[Descriptors Section]
[Threads Section]
  [Thread 1]
  [Thread 2]
  ...
  [Thread N]
[End Signature]
[Bookmarks Section] (опционально, только в версии 2.1.0+)
[End Signature]
```

## Константы

```go
Signature = 0x45617379  // "Easy" в ASCII
MinCompatibleVersion = 0x00010000  // v0.1.0
Version100 = 0x01000000  // v1.0.0
Version130 = 0x01030000  // v1.3.0
Version200 = 0x02000000  // v2.0.0
Version210 = 0x02010000  // v2.1.0
```

## Заголовок файла (Header)

### Версия < 1.3.0

```
[uint32] SIGNATURE = 0x45617379
[uint32] VERSION
[uint32] PID (process ID)
[int64]  CPU_FREQUENCY
[uint64] BEGIN_TIME (timestamp)
[uint64] END_TIME (timestamp)
[uint32] BLOCKS_COUNT
[uint64] MEMORY_SIZE
[uint32] DESCRIPTORS_COUNT
[uint64] DESCRIPTORS_MEMORY_SIZE
```

### Версия 1.3.0 - 1.x.x

```
[uint32] SIGNATURE = 0x45617379
[uint32] VERSION
[uint64] PID (расширено с uint32 до uint64)
[int64]  CPU_FREQUENCY
[uint64] BEGIN_TIME
[uint64] END_TIME
[uint32] BLOCKS_COUNT
[uint64] MEMORY_SIZE
[uint32] DESCRIPTORS_COUNT
[uint64] DESCRIPTORS_MEMORY_SIZE
```

### Версия 2.0.0 - 2.0.x

```
[uint32] SIGNATURE = 0x45617379
[uint32] VERSION
[uint64] PID
[int64]  CPU_FREQUENCY
[uint64] BEGIN_TIME
[uint64] END_TIME
[uint64] MEMORY_SIZE (переставлено)
[uint64] DESCRIPTORS_MEMORY_SIZE (переставлено)
[uint32] BLOCKS_COUNT (переставлено)
[uint32] DESCRIPTORS_COUNT (переставлено)
```

### Версия 2.1.0+

```
[uint32] SIGNATURE = 0x45617379
[uint32] VERSION
[uint64] PID
[int64]  CPU_FREQUENCY
[uint64] BEGIN_TIME
[uint64] END_TIME
[uint64] MEMORY_SIZE
[uint64] DESCRIPTORS_MEMORY_SIZE
[uint32] BLOCKS_COUNT
[uint32] DESCRIPTORS_COUNT
[uint32] THREADS_COUNT (новое поле)
[uint16] BOOKMARKS_COUNT (новое поле)
[uint16] PADDING (должно быть 0)
```

## Секция дескрипторов (Descriptors Section)

Повторяется `DESCRIPTORS_COUNT` раз:

```
[uint16] SIZE (размер всего дескриптора в байтах)
[BaseBlockDescriptor]
  [uint32] ID
  [int32]  LINE (номер строки в исходном файле)
  [uint32] COLOR (ARGB цвет)
  [uint8]  TYPE (BlockType: Event=0, Block=1, Value=2)
  [uint8]  STATUS
[uint16] NAME_LENGTH (включая завершающий \0)
[byte*]  NAME (NAME_LENGTH байт, null-terminated строка)
[byte*]  FILE (оставшиеся байты до SIZE, null-terminated строка)
```

**Расчет размера FILE:**
```
fileSize = SIZE - (4+4+4+1+1+2+NAME_LENGTH)
         = SIZE - 16 - NAME_LENGTH
```

## Секция потоков (Threads Section)

Каждый поток имеет структуру:

```
[uint64] THREAD_ID
[uint16] THREAD_NAME_SIZE
[byte*]  THREAD_NAME (THREAD_NAME_SIZE байт)

[Context Switches Subsection]
  [uint32] CONTEXT_SWITCHES_COUNT
  For each context switch:
    [uint16] SIZE
    [ContextSwitch]
      [uint64] THREAD_ID
      [uint64] BEGIN_TIME
      [uint64] END_TIME
      [byte*]  NAME (остаток до SIZE, null-terminated)

[Blocks Subsection]
  [uint32] BLOCKS_COUNT (для этого потока)
  For each block:
    [uint16] SIZE
    [Block]
      [uint64] BEGIN_TIME
      [uint64] END_TIME
      [uint32] ID (ссылка на дескриптор)
      [byte*]  NAME (остаток до SIZE, null-terminated, опционально)
```

**Важно:** Блоки могут быть вложенными (иметь дочерние блоки). Формат поддерживает рекурсивную структуру.

### Конец секции потоков

```
[uint64] END_SIGNATURE (первые 4 байта = 0x45617379)
```

Для проверки читаем uint64, если младшие 4 байта равны сигнатуре - это конец секции.

## Секция закладок (Bookmarks Section)

Только для версии 2.1.0+ и если `BOOKMARKS_COUNT > 0`:

```
For each bookmark (BOOKMARKS_COUNT раз):
  [uint16] SIZE
  [uint64] POSITION
  [uint32] COLOR
  [byte*]  TEXT (SIZE - 12 байт, null-terminated)

[uint32] END_SIGNATURE = 0x45617329
```

## Типы данных

### BlockType (uint8)

```
Event = 0  // Событие (точка во времени)
Block = 1  // Блок выполнения (интервал времени)
Value = 2  // Значение (для произвольных данных)
```

### Timestamp

- Тип: `uint64`
- Единицы: наносекунды
- Если `CPU_FREQUENCY != 0`, может требоваться конвертация из тактов CPU

### Process/Thread ID

- PID: `uint32` (< v1.3.0) или `uint64` (>= v1.3.0)
- Thread ID: `uint64`

## Особенности парсинга

### 1. Упаковка структур

Все структуры используют `#pragma pack(push, 1)` - нет выравнивания, все поля следуют подряд.

### 2. Endianness

Little-endian для всех многобайтовых значений.

### 3. Null-terminated строки

Все строки завершаются нулевым байтом (`\0`). При чтении строки, последний байт нужно отбросить.

### 4. Переменная длина записей

Каждая запись (дескриптор, блок, переключение контекста, закладка) начинается с `uint16 SIZE`, указывающего полный размер записи в байтах.

### 5. Рекурсивные блоки

Блоки могут содержать дочерние блоки. При парсинге нужно поддерживать стек для корректной обработки вложенности.

### 6. Валидация

- Проверяйте сигнатуру в начале файла
- Проверяйте версию на совместимость (>= MIN_COMPATIBLE_VERSION)
- Проверяйте конечные сигнатуры секций

## Примеры

### Минимальный файл (версия 2.1.0)

```
45 61 73 79              // Signature "Easy"
00 00 01 02              // Version 2.1.0
D2 04 00 00 00 00 00 00  // PID = 1234
00 00 00 00 00 00 00 00  // CPU_FREQUENCY = 0
10 00 00 00 00 00 00 00  // BEGIN_TIME = 16
20 00 00 00 00 00 00 00  // END_TIME = 32
00 00 00 00 00 00 00 00  // MEMORY_SIZE = 0
00 00 00 00 00 00 00 00  // DESCRIPTORS_MEMORY_SIZE = 0
00 00 00 00              // BLOCKS_COUNT = 0
00 00 00 00              // DESCRIPTORS_COUNT = 0
00 00 00 00              // THREADS_COUNT = 0
00 00                    // BOOKMARKS_COUNT = 0
00 00                    // PADDING = 0
79 73 61 45 00 00 00 00  // END_SIGNATURE
```

### Пример дескриптора

```
2A 00                    // SIZE = 42 bytes
01 00 00 00              // ID = 1
7B 00 00 00              // LINE = 123
FF 00 00 FF              // COLOR = 0xFF0000FF (синий)
01                       // TYPE = Block
00                       // STATUS = 0
09 00                    // NAME_LENGTH = 9
4D 79 42 6C 6F 63 6B 00  // NAME = "MyBlock\0"
6D 61 69 6E 2E 63 70 70 00  // FILE = "main.cpp\0"
```

### Пример блока

```
1C 00                    // SIZE = 28 bytes
00 10 00 00 00 00 00 00  // BEGIN = 4096
00 20 00 00 00 00 00 00  // END = 8192
01 00 00 00              // ID = 1 (ссылка на дескриптор)
54 65 73 74 00           // NAME = "Test\0" (опционально)
```

## Ссылки на исходный код

В кодовой базе EasyProfiler:

- `easy_profiler_core/writer.cpp` - сериализация
- `easy_profiler_core/reader.cpp` - десериализация
- `easy_profiler_core/include/easy/serialized_block.h` - определения структур
- `easy_profiler_core/include/easy/profiler_public_types.h` - публичные типы

## История изменений формата

| Версия | Изменения |
|--------|-----------|
| 0.1.0  | Первая версия |
| 1.0.0  | Стабилизация формата |
| 1.3.0  | Thread ID: uint32 → uint64, PID: uint32 → uint64 |
| 2.0.0  | Изменен порядок полей в заголовке |
| 2.1.0  | Добавлены THREADS_COUNT, BOOKMARKS_COUNT и секция закладок |

## Совместимость

Парсер должен поддерживать чтение всех версий >= 0.1.0. При чтении заголовка сначала читайте сигнатуру и версию, затем используйте version-specific логику для чтения остальных полей.
