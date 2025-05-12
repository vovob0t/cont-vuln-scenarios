FROM golang:1.18-alpine
WORKDIR /app

ENV ADMIN_PSWD="SECRET-TOKEN-420"

# Нужен git для go mod
RUN apk add --no-cache git ca-certificates bash

# Скачиваем модули
# COPY go.mod go.sum ./
COPY . .
RUN go mod tidy
RUN go mod download

# Копируем остальной код (config.yaml, main.go)
# COPY . .

# Собираем бинарь
RUN go build -o server main.go

EXPOSE 8080
CMD ["./server"]
