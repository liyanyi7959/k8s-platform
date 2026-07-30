# Backend module map

`internal` is organized by DDD bounded context. New business behavior belongs to
one of the context directories below and uses a vertical slice:

```text
<context>/
  domain/          entities, value objects, invariants, domain events
  application/     commands, queries and transaction orchestration
  ports/           interfaces required by the application layer
  adapters/
    http/          HTTP DTOs and handlers
    mysql/         persistence implementations
    redis/         cache, ticket and coordination implementations
    kubernetes/    Kubernetes integration implementations
```

Every bounded context has a catalog package so ownership is discoverable from
the directory tree. A context containing only `doc.go` is explicitly legacy;
only contexts with `domain`, `application`, `ports` or `adapters` contain
migrated behavior.

## Bounded contexts

| Context | Owns | Current migration state |
| --- | --- | --- |
| `iam` | identity, roles, permissions, authentication policy | legacy |
| `workspace` | projects, tenancy, namespace assignment | legacy |
| `fleet` | cluster registry, connectivity, fleet inventory | legacy |
| `kops` | Kubernetes resources, diagnostics, observability reads | legacy |
| `incident` | alerts, incidents, timeline and lifecycle | vertical slice |
| `ai` | providers, conversations, diagnosis and suggestions | legacy |
| `change` | proposals, approval policy, execution authorization | vertical slice in progress |
| `provisioning` | hosts, plans, preflight and cluster delivery | legacy |
| `audit` | immutable actor and operation evidence | legacy |
| `platform` | platform-wide settings and technical administration | legacy |

Application templates belong to `provisioning`: they describe delivery inputs,
not platform-wide configuration.

## Transitional directories

`legacy/controller`, `legacy/service`, and `legacy/model` are compatibility
packages for code not yet migrated. They are not valid destinations for new
domain behavior. A migrated slice keeps its v1 handler as an adapter and routes
both v1 and v2 through the context application layer before its old files are
removed.

`router` is the HTTP composition root during the modular-monolith phase. It may
wire every context, but it must not contain business rules. `middleware`, `db`,
`auth`, and `config` provide shared technical capabilities rather than business
models.

## Cross-context orchestration

`orchestration/` is the explicit, narrow boundary for workflows that must
coordinate more than one bounded context. Its packages are named by workflow
(`ai`, `kops`, and `provisioning`) and are composed only from the router.
Examples include AI action execution through Kops and Change, deployment
execution across Provisioning, Fleet and Platform, and Kops permission-audit
workflows.

It is not a general implementation directory: a component that serves one
bounded context belongs under that context's `adapters/` tree. The former
`integration/` directory is intentionally empty and is protected by an
architecture test so it cannot become another legacy service layer.

## Dependency direction

```text
router/composition -> adapters -> application -> domain
                                      |
                                      v
                                    ports
                                      ^
                                      |
                                   adapters
```

- `domain` imports only the Go standard library.
- `application` may import its own `domain` and `ports`, never adapters or legacy layers.
- An adapter may implement its context's ports, but cannot import another context's adapters.
- Cross-context collaboration uses an explicit public port or integration event.
- A temporary cross-context workflow lives only in its named
  `orchestration/<workflow>` package and must be decomposed into ports/events
  before it can become single-context behavior.
- Database tables have one owning context even while all contexts share one MySQL instance.

## Migration rule

All pre-DDD horizontal packages live below `internal/legacy`; there are no
top-level `controller`, `service`, or `model` packages. Existing behavior stays
available while each context replaces its legacy implementation vertically.
Architecture tests freeze the legacy file-count baseline so this area can only
shrink during migration.
Architecture tests prevent completed slices from importing the legacy area.
