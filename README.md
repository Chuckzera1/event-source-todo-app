TODO LIST

This is an API to help me study abou event source and golang.

The idea is to use events to populate the todo list using pub/sub qeues and events.

We are using Clean/Hexa Arch and SOLID principle to write this API.
(Thinking about going all the way with Clean Arch, but can be tricky with golang sometimes)

## Contributing / Agent Guidelines

See [CLAUDE.md](CLAUDE.md) for architecture rules, design patterns, Go conventions, testing
strategy, and the anti-patterns checklist that all code in this repo must follow.

## Run with Docker Compose

Start API + Postgres:

```bash
docker compose up --build
```

Healthcheck endpoint:

```bash
curl http://localhost:8080/health
```

POC default Postgres credentials in `docker-compose.yml`:

- Database: `todo`
- User: `todo`
- Password: `todo`
