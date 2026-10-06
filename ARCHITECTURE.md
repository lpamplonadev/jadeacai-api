# Architecture

The backend uses a layered structure. The dependency direction is inward: HTTP transport calls application use cases; application depends on domain rules and repository ports; infrastructure implements those ports. `cmd/api` is the composition root.

## Layers

- `cmd/api`: loads environment configuration, opens PostgreSQL, wires the service and router, and starts the server.
- `internal/domain`: framework-independent business types and rules such as order statuses, catalog kinds, and UUID validation.
- `internal/application`: use-case models, repository ports, and order/catalog operations. Business validation, catalog key generation, and defaults belong here.
- `internal/infrastructure/postgres`: database connection and SQL repository implementation. It implements `application.Repository` and does not register routes.
- `internal/transport/httpapi`: Gin router, middleware, request binding, query parsing, HTTP status mapping, and response serialization.

## Change Rules

- Keep route paths, JSON field names, and status codes stable unless the API contract is intentionally changed.
- Keep Gin types out of application and domain packages; keep SQL out of transport handlers.
- Put persistence-specific tests beside the PostgreSQL adapter and HTTP contract tests beside the router.
- Wire new implementations in `cmd/api`; do not construct repositories inside handlers.

Run `go test ./...` after backend changes.
