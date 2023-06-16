# LogDoc File Watcher

## Описание утилиты

Существует много приложений, оборудования, которые пишут логи своей работы в файлы. Утилита LogDoc File Watcher предназначена для мониторинга одного или нескольких таких файлов, с передачей данных на сервер LogDoc.

## Config Reference

```json
{
  "debug": true, // Запуск встроенного профилировщика приложения, доступен по адресу http://localhost:6060/debug/pprof/
  "logdoc": {
    "host": "1.1.1.1", // LogDoc Server host
    "port": "5656",    // LogDoc Server port
    "proto": "tcp",    // LogDoc appender protocol (tcp/udp)
    "default": {
      "app": "file-watcher", // Приложение по умолчанию, которое будет отображаться в интерфейсе LogDoc (если не будет переопределено на уровне файла ниже)
      "source": "file-watcher source", // Источник по умолчанию, так же будет отображаться в интерфейсе LogDoc (если не будет переопределено на уровне файла ниже)
      "level": "DEBUG" // Уровень лога, так же будет отображаться в интерфейсе LogDoc (если не будет переопределено на уровне файла ниже)
    }
  },
  "files": [ // массив обрабатываемых файлов (ниже разберем подробно структуры)
    {
      "path": "/var/log/nginx/access.log",
      "pattern": "%{IP:client} %{USER:ident} %{USER:auth} \\[%{HTTPDATE:timestamp}\\] \"%{WORD:method} %{URIPATHPARAM:request}",
      "app": "file-watcher nginx access.log",
      "source": "file-watcher nginx access.log source",
      "level": "INFO",
      "layout": "02/Jan/2006:15:04:05 -0700"
    },
    {
      "path": "/var/log/daemon.log",
      "pattern": "%{CUSTOM_DATE:timestamp} %{WORD:host} %{WORD:process}\\[%{NUMBER:pid}\\]: %{GREEDYDATA:message}.",
      "custom": "%{MONTH} %{MONTHDAY} %{TIME}",
      "app": "file-watcher daemon.log",
      "layout": "Jan 02 15:04:05"
    },
    {
      "path": "/var/log/fontconfig.log",
      "pattern": "%{GREEDYDATA:path}: %{GREEDYDATA:message}",
      "app": "file-watcher fontconfig.log",
      "layout": "02/Jan/2006:15:04:05 -0700"
    },
    {
      "path": "/var/log/dpkg.log",
      "pattern": "(?P<timestamp>%{YEAR}-%{MONTHNUM}-%{MONTHDAY} %{TIME}) %{DATA:message}:%{DATA:arch} %{GREEDYDATA:version}",
      "app": "file-watcher dpkg.log",
      "layout": "2006-01-02 15:04:05"
    }
  ]
}
```

Допустим мы хотим мониторить изменения в 4х разных системных журналах, разберем структуру каждого файла отдельно

Всегда необходимо знать, файлы како структуры будут записаны 

### Nginx Access Log

Примеры данных, которые будут записаны в этот журнал:

178.215.145.104 - - [09/Jun/2023:14:09:30 +0300] "GET /api/stream/33df6bb366ab6d6bfe50d9ea3246482c HTTP/1.1" 403 16 "-" "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 YaBrowser/23.1.1.1114 Yowser/2.5 Safari/537.36"

127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207

Структура обработки данного файла может выглядеть так:

```json
{
  "path": "/var/log/nginx/access.log", // Путь к файлу в системе
  "pattern": "%{IP:client} %{USER:ident} %{USER:auth} \\[%{HTTPDATE:timestamp}\\] \"%{WORD:method} %{URIPATHPARAM:request}", // grok паттерн для разбора строки на поля с данными (ниже рассмотрим примеры паттернов с их подробным описанием)
  "app": "file-watcher nginx access.log", // переопределяем на уровне файла имя приложения LogDoc
  "source": "file-watcher nginx access.log source", // переопределяем на уровне файла источник данных для LogDoc
  "level": "INFO", // переопределяем на уровне файла уровень лога записи в LogDoc
  "layout": "02/Jan/2006:15:04:05 -0700" // Шаблон даты для корректного форматирования timestamp данных из файла, как видно из примера строк лога, дата приходит в формате [09/Jun/2023:14:09:30 +0300]
}
```

### Логи фоновых процессов(daemons) daemon.log

Примеры данных, записываемых в этот файл:

Jun 15 21:04:48 hetzner systemd[980454]: Reached target Exit the Session.
Jun 15 21:04:48 hetzner systemd[1]: user@1001.service: Succeeded.
Jun 15 21:04:48 hetzner systemd[1]: Stopped User Manager for UID 1001.
Jun 15 21:04:48 hetzner systemd[1]: Stopping User Runtime Directory /run/user/1001...

```json    {
{
  "path": "/var/log/daemon.log", // Путь к файлу в системе
  "pattern": "%{CUSTOM_DATE:timestamp} %{WORD:host} %{WORD:process}\\[%{NUMBER:pid}\\]: %{GREEDYDATA:message}.", // grok паттерн для разбора строки на поля с данными (ниже рассмотрим примеры паттернов с их подробным описанием)
  "custom": "%{MONTH} %{MONTHDAY} %{TIME}", // custom grok паттерн, можно указать его, при отсутсвии подходящего grok паттерна, либо сделать6 как в примере с файлом dpkg.log ниже (timestamp)
  "app": "file-watcher daemon.log", // переопределяем на уровне файла имя приложения LogDoc
  "layout": "Jan 02 15:04:05" // Шаблон даты для корректного форматирования даты по умолчанию
}    
```

### FontConfig

Это библиотека, разработанная для предоставления списка доступных шрифтов приложениям, а также для настройки того, как шрифты будут отображены, ее логи будут записаны в файл fontconfig.log

Примеры данных, которые будут записаны в данный log файл:

/usr/share/fonts/truetype/noto: skipping, looped directory detected
/usr/share/fonts/type1/gsfonts: skipping, looped directory detected
/usr/share/fonts/type1/urw-base35: skipping, looped directory detected
/usr/share/fonts/X11/encodings/large: skipping, looped directory detected
/var/cache/fontconfig: cleaning cache directory

```json
{
  "path": "/var/log/fontconfig.log", // Путь к файлу в системе
  "pattern": "%{GREEDYDATA:path}: %{GREEDYDATA:message}", // grok паттерн для разбора строки на поля с данными (ниже рассмотрим примеры паттернов с их подробным описанием)
  "app": "file-watcher fontconfig.log", // переопределяем на уровне файла имя приложения LogDoc
  "layout": "02/Jan/2006:15:04:05 -0700" // в данном файле дата отсутствует, LogDoc же требует наличия даты источника лога, укажем шаблон даты для корректного форматирования даты по умолчанию
}
```

### Логи установки пакетов будут записаны в dpkg.log

Примеры данных, которые будут записаны в данный log файл:

2023-06-06 14:42:33 status half-configured libc-bin:amd64 2.31-13+deb11u6
2023-06-06 14:42:33 status installed libc-bin:amd64 2.31-13+deb11u6
2023-06-06 14:42:33 status half-configured libc-bin:amd64 2.31-13+deb11u6

```json
{
  "path": "/var/log/dpkg.log", // Путь к файлу в системе
  "pattern": "(?P<timestamp>%{YEAR}-%{MONTHNUM}-%{MONTHDAY} %{TIME}) %{DATA:message}:%{DATA:arch} %{GREEDYDATA:version}", // подходящего grok паттерна для разбора даты нет, сделаем свой объединенный паттерн из нескольких отдельных токенов строки %{YEAR}-%{MONTHNUM}-%{MONTHDAY} %{TIME} (ниже рассмотрим примеры паттернов с их подробным описанием)
  "app": "file-watcher dpkg.log", // переопределяем на уровне файла имя приложения LogDoc
  "layout": "2006-01-02 15:04:05" // Шаблон даты для корректного чтения даты поля timestamp
}
```

# Grok Patterns

Документация по использованию Grok Patterns

Grok - это мощный инструмент для анализа и извлечения данных из неструктурированных логов. Он использует шаблоны, называемые Grok Patterns, для определения структуры логов и извлечения значений из них. Здесь мы рассмотрим основы использования Grok Patterns и примеры.

Единица Grok Pattern представляет собой регулярное выражение, либо составной элемент из нескольких grok patterns

1. Основы Grok Patterns

Grok Pattern - это шаблон, который описывает структуру строки и определяет, какие значения нужно извлечь. Шаблон состоит из нескольких частей:

\- %{PATTERN_NAME}: это часть шаблона, которая соответствует конкретному типу данных. Например, %{WORD} описывает строку, состоящую только из букв и цифр.

\- [PATTERN_MODIFIER]: это модификатор, который применяется к шаблону. Например, + означает, что шаблон должен соответствовать одному или более экземплярам данного паттерна.

2. Примеры Grok Patterns

Ниже приведены несколько примеров Grok Patterns с объяснениями.

\- %{IPORHOST:clientip} - это шаблон, который извлекает IP-адрес клиента из лога. Он соответствует любому IP-адресу или доменному имени, которое может быть указано в логе. Значение извлекается и сохраняется в поле clientip.

IPORHOST представляет собой комбинацию полей (?:%{IP}|%{HOSTNAME}) Этот Grok-шаблон представляет собой именованный (IPORHOST) захватывающий блок, который соответствует либо IP-адресу, либо имени хоста. В зависимости от того, какие данные содержатся в логах, этот шаблон может использоваться для извлечения соответствующих значений из строк логов. Например, если в логах содержатся IP-адреса и имена хостов, то этот шаблон может помочь извлечь эти данные и сохранить их в соответствующих полях.

Захватывающий блок - это часть регулярного выражения, которая используется для захвата определенной группы символов. В данном случае, захватывающий блок представлен символами "(?)" и используется для захвата IP-адреса или имени хоста из строки логов.

Символ "?" в регулярных выражениях обозначает необязательность предшествующего символа или группы символов. В данном случае, он указывает на то, что захватывающий блок может содержать либо IP-адрес, либо имя хоста.

\- %{TIMESTAMP_ISO8601:timestamp} - это шаблон, который извлекает дату и время из лога. Он соответствует формату даты и времени ISO 8601, который часто используется в логах. Значение извлекается и сохраняется в поле timestamp. Pattern TIMESTAMP_ISO8601, внутри представляет собой набор паттернов %{YEAR}-%{MONTHNUM}-%{MONTHDAY}[T ]%{HOUR}:?%{MINUTE}(?::?%{SECOND})?%{ISO8601_TIMEZONE}? 

\- %{WORD:method} %{URIPATHPARAM:request} HTTP/%{NUMBER:httpversion} - это шаблон, который извлекает информацию о запросе HTTP из лога. Он соответствует методу запроса (GET, POST и т.д.), пути запроса и версии HTTP. Значения извлекаются и сохраняются в поля method, request и httpversion соответственно. WORD внутри является регулярным выражением: \b\w+\b

Заключение

Grok Patterns - это мощный инструмент для анализа и извлечения данных из неструктурированных логов. Они позволяют определить структуру лога и извлечь нужные значения.

Множество предопределенных grok паттернов можно посмотреть здесь https://github.com/vjeantet/grok/tree/master/patterns
