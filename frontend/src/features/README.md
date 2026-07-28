# Frontend feature map

Frontend business code is grouped by the same bounded contexts as the backend:

```text
features/<context>/
  pages/   routed screens owned by the context
  api/     HTTP contracts and request functions
  types/   context-owned DTO and view-model types
```

`shared` contains technical reuse only. Components, hooks, layouts, theme and
utilities remain top-level shared infrastructure. Route URLs are stable; only
their source component paths changed.

The current contexts are `iam`, `workspace`, `fleet`, `kops`, `incident`, `ai`,
`provisioning`, and `platform`. Change proposal UI currently lives with AI until
it receives an independent workflow screen, but its backend authorization is
owned by the Change context.
