# Decisions

| Area | Decision | Basis |
| --- | --- | --- |
| iOS spelling | Use `ios-<resource>`. | Explicit user choice. |
| Version selection | Do not equate matching nouns or generation ranks with compatibility. | Generator currently compares names, not behavior. |
| Writes | Do not infer write compatibility from GET comparisons. | Request bodies and side effects can differ independently. |
| Cross-platform apps | Keep tenant-app APIs platform-neutral. | Tenant app schemas include Windows support. |
| Seamless onboarding | Keep platform-neutral. | Request schema explicitly supports Apple, Linux, and Windows. |
| Scoped pipeline variants | Retain creation/attachment, definition/execution, and resource/detachment distinctions. | Different routes and effects. |
| Historical reports | Preserve old CSV and evidence. | They record the previously tested binary, not the new CLI tree. |

## Endpoint decisions

| Pair | Evidence | Decision |
| --- | --- | --- |
| Legacy/v1 installed devices for an app version | Identical complete two-record live responses; identical normalized operation schemas, including parameters and responses. | Retire the legacy command; use `installdevice list` and v1. |
| Legacy/v1 application collections and versions | Collections total 424 versus 752 with different fields; selected app/version exists only through v1. Legacy version list empty versus one v1 result. | Keep both; not interchangeable. |
| Legacy/v2 device and device-request details | Different envelopes, fields, types, and management data for the same device. | Keep all. |
| Device application lists and iOS catalog | Legacy device inventory has six apps; newer device-app API returns zero for this Android device; tenant iOS catalog has 21 records. | Keep distinct; prefix only the iOS catalog. |
| Enterprise application catalog versus tenant-app catalog | Selected collections total 752 versus nine with different fields and platform contracts. | Keep both; tenant-app supports multiple platforms. |
| Legacy/v1 event feeds | Same sampled records; different next URL. Normalized contracts also differ in pagination default and required response fields. | Keep both. |
| v1/v2 status metrics | Different active/inactive/last-seen/unprovisioned fields. | Keep both. |
| v0/new geofences | Different response envelopes and request filters, enterprise versus tenant scoping. | Keep both. |
| Enterprise/new command collections | Three versus two records for the same device/command filters; different envelopes and filters. | Keep both. |
| Command status APIs | Same request can be read, but response envelopes and selectors differ. | Keep both. |
| Latest device event versus command status | Event and command-status schemas and identifier types differ. | Keep both. |
| Legacy user versus authn user | Different envelopes, identifiers, and identity fields. | Keep both. |
| Blueprint generations | Legacy group-scoped templates/revisions versus tenant modern blueprints/versions. Prior live legacy create failed without a returned ID; modern lifecycle succeeded. | Keep both; no equivalent legacy fixture for a matched-item comparison. |
| Pipeline scoped variants | Definitions versus executions, owned versus attached lists, and create/delete versus attach/detach are different operations. | Keep all variants. |

No write endpoints are removed based on these read-only comparisons. No new API
writes or approval requests were issued by this task.
