FROM golang:1.24.2-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
ENV MIGRATE_VERSION=v4.18.3
RUN apk add --no-cache wget tar && \
    wget https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz -O - | tar -xz && \
    mv migrate /app/migrate_binary
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/user-service ./cmd/user-service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/migrate_binary /usr/local/bin/migrate
COPY --from=builder /app/user-service /app/user-service
CMD [ "/app/user-service" ]
