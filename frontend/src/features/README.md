# Frontend feature map

Frontend business code is grouped by the same bounded contexts as the backend:

```text
features/<context>/
  index.ts public API consumed by other contexts
  pages/   routed screens owned by the context
  api/     HTTP contracts and request functions
  types/   context-owned DTO and view-model types
  schemas/ context-owned validation contracts
  components/ context-owned UI with domain behavior
```

`shared` contains technical reuse only. Components, hooks, layouts, theme and
utilities remain top-level shared infrastructure. Route URLs are stable; only
their source component paths changed.

Cross-context imports must use `@/features/<context>` and cannot reach into
another context's `api`, `types`, `schemas`, `components` or `pages` folders.
Top-level technical infrastructure cannot import a feature. App composition
files and Umi models may consume feature public APIs.

The current contexts are `iam`, `workspace`, `fleet`, `kops`, `incident`, `ai`,
`provisioning`, and `platform`. Change proposal UI currently lives with AI until
it receives an independent workflow screen, but its backend authorization is
owned by the Change context.
