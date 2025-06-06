# oip-course

Репозиторий по домашним заданиям курса ОИП.

Автор репозитория: Мударисов Ренат 11-101

## Задания

### Предусловия

1. Установка Go

Установить Go можно с официального сайта: https://go.dev/doc/install

2. Установка зависимостей

В корневой директории проекта выполните команду в терминале:
```
go mod download
```

### Задание 1. Краулер

Для запуска краулера в корневой директории выполните команду в терминале:
```
go run cmd/crawler/main.go
```

### Задание 2. Токенайзер

Для запуска токенайзера в корневой директории выполните команду в терминале:
```
go run cmd/tokenizer/main.go
```

### Задание 3. Инвертированный индекс

1. Для создания инвертированного индекса в корневой директории выполните команду в терминале:
```
go run cmd/inverted_index_builder/main.go
```

2. Для запуска булевого поиска по индексу в корневой директории выполните команду в терминале:
```
go run cmd/inverted_index_search/main.go
```

### Задание 4. TF-IDF

Для запуска вычисления TF-IDF в корневой директории выполните команду в терминале:
```
go run cmd/tf_idf/main.go
```