FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/todo-api ./cmd/todo-api

FROM alpine:3.22 AS final

RUN addgroup -S app && adduser -S app -G app
USER app

WORKDIR /app
COPY --from=builder /bin/todo-api /app/todo-api

EXPOSE 8080

CMD ["/app/todo-api"]
