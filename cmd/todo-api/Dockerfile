# Build from repository root:
#   docker build -f cmd/todo-api/Dockerfile -t todo-api .
FROM golang:1.26-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/todo-api ./cmd/todo-api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/todo-api /todo-api

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/todo-api"]
