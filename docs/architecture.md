# Go Comet Agent Platform

This backend starts with Gin at the HTTP boundary and keeps Gin out of app,
domain, repository, and infrastructure packages. The first phase focuses on
the API core: auth, model configuration, chat SSE, Agent tool orchestration,
RAG boundaries, memory extraction boundaries, async tasks, and deployment
storage services.

## Boundaries

- `internal/transport/http`: Gin router, middleware, request DTOs, response envelope.
- `internal/app`: use cases and service orchestration using `context.Context`.
- `internal/domain`: framework-free business contracts.
- `internal/repository`: persistence interfaces and generated SQL access.
- `internal/infrastructure`: LLM, Agent, RAG, Memory, Queue, Storage, Realtime adapters.

## First Acceptance Path

1. Start storage services from `deployments/docker-compose.yml`.
2. Run migrations with the future goose runner.
3. Start `cmd/api`.
4. Register/login, configure models, upload content, stream chat events.

