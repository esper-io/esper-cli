# x-esper-* overlay schema

Overlay annotations added to the canonical OpenAPI files under `spec/openapi/`. Applied at the operation level unless noted. The codegen and contract checker consume these; treat missing-required-annotation as a contract-check failure.

## Operation-level (required on every operation)

| Field | Type | Meaning |
|---|---|---|
| `x-esper-destructive` | bool | true → generated command requires confirmation or `--yes`. Applies to wipe, delete, remove-user, factory-ish operations. Default must be explicit, never omitted. |
| `x-esper-pagination` | enum: `none` \| `limit-offset` \| `apps-envelope` | Which envelope unwrapper the runtime uses. `limit-offset` = `{limit, offset, count, next, previous, results}`; `apps-envelope` = `{code, message, content}`. |
| `x-esper-verb` | string | CLI verb this operation maps to (`list`, `get`, `create`, `update`, `delete`, or a domain verb like `reboot`). Codegen uses this, not HTTP-method heuristics. |
| `x-esper-noun` | string | CLI noun (command group), singular kebab-case (`device`, `blueprint`, `tenant-app`). |

## Operation-level (optional)

| Field | Type | Meaning |
|---|---|---|
| `x-esper-response-envelope` | enum: `apps-envelope` | Successful JSON responses are wrapped in `{code,message,content}`. `--json` preserves the raw envelope; human output renders `content`. Derived service-wide when the service has an `apps-envelope` list. |
| `x-esper-require-one-of` | list of parameter names | At least one named query/header flag must be supplied. The flags are not mutually exclusive. Zero values is a usage error (exit 2) before any API request. |
| `x-esper-hidden` | bool | Operation exists in spec but gets no generated command (deprecated or device-facing). |
| `x-esper-scope-parent` | string | Present on a parent-scoped collection operation; names the parent noun whose path parameter becomes the scope flag. Triggers the "Scoped collections" merge rule in conventions.md. |
| `x-esper-examples` | list | `{description, command, response-fixture}` triples; fixture is a path under `spec/fixtures/`. Used for docs, `/esper` skill generation, and golden tests. |

## Parameter-level (optional)

| Field | Type | Meaning |
|---|---|---|
| `x-esper-name-resolvable` | object `{resource, lookup-endpoint}` | Flag accepts a human name; runtime resolves to ID via the given list endpoint (e.g. device name → device id). |

## Spec-file-level (info section)

| Field | Type | Meaning |
|---|---|---|
| `x-esper-generation` | string | `v0`, `v1`, `v2`, `pipelines-v0`, `authn2`, ... |
| `x-esper-auth` | string | Always `bearer` for public APIs (JWT or PAT). |
