# Crawler CLI

CLI на Go обходит список стартовых URL и сохраняет дерево найденных HTML-страниц в JSON.

## Сборка

```bash
go build -o crawler-cli .
```

На Windows появится файл `crawler-cli.exe`.

## Запуск

```bash
crawler-cli \
  --urls https://google.com,https://example.com \
  --depth 3 \
  --timeout 2m \
  --request-timeout 10s \
  --output result.json \
  --log crawler.log
```

В PowerShell:

```powershell
.\crawler-cli.exe --urls https://google.com,https://example.com --depth 3 --timeout 2m --request-timeout 10s --output result.json --log crawler.log
```

## Флаги

| Флаг | По умолчанию | Описание |
| --- | --- | --- |
| `--urls` | нет, обязателен | стартовые URL через запятую |
| `--depth` | `1` | сколько уровней ссылок обходить ниже стартовой страницы. `0` сохраняет только стартовые URL |
| `--timeout` | `1m` | общий срок обхода |
| `--request-timeout` | `10s` | срок одного HTTP-запроса |
| `--output` | `result.json` | дерево страниц в JSON |
| `--log` | `crawler.log` | ошибки и HTTP-статусы |

Без `--urls` программа завершается с кодом 2 и печатает справку.

## Поведение

Одновременно выполняется не больше 10 запросов. Ссылка обходится, только если её хост совпадает с хостом своего стартового URL. Сравнение URL идёт по строке как она получена, без нормализации.

Редирект, ответ не `200`, не-HTML и недоступная страница пропускаются и пишутся в лог. Ошибка одного URL не останавливает остальные. Повторный запрос по той же строке URL не выполняется.

Ctrl+C останавливает обход. Уже собранное дерево записывается в `--output`.

## Тесты

```bash
go test ./...
```
