# Canonical OpenAPI specifications

These resolved OpenAPI 3.1 files are generated from the private
`github.com/esper-code/esper-api-docs` repository at revision
`d5563a0e35361f9c0dd0aefec140f6071619e8a1` and, when supplied, the deployed
public Redocly OAS snapshot. The private source revision is currently behind
the public reference, so the snapshot contributes documented public routes
that are absent from the checkout. The generated manifest records the public
snapshot hash and path count for review.

The upstream assembly joins 12 platform service specifications, rewrites
cross-service references, prefixes colliding tags and components, and splits
the result into 205 path files. `tools/specbundle/bundle.sh` bundles that split
source and partitions it into API generations without external references.

Two source paths are intentionally excluded:

- `/device/v0/devices/` is deprecated in the device API public specification.
- `/sys/health` is an operational health endpoint, not a customer API.

Run the following from an authenticated local checkout. The required public OAS
argument is fetched only during this build step; released commands never fetch
or interpret OpenAPI documents at runtime.

```sh
tools/specbundle/bundle.sh /path/to/esper-api-docs \
  https://api.esper.io/page-data/shared/oas-openapi.yaml.json
npx --yes @redocly/cli@1.34.5 lint --config spec/redocly.yaml
```

`manifest.json` records the source, public snapshot provenance, excluded-path
reasons, emitted, and per-generation path counts. Generated specifications are
JSON-formatted YAML so downstream Go tools can parse them with the standard
library.

## Public operation boundary

The bundler emits only operations present in the supplied public Redocly OAS.
It normalizes `/api` prefixes and trailing slashes before comparison, removes
any platform-only operation present in an older overlay, and records the public
operation keys in `manifest.json`. `tools/contractcheck` rejects a generated
operation missing from that manifest.

`secureadb` is a hand-written Python-parity command and the sole exception to
the generated API-operation boundary. Its Remote ADB calls are internal to the
command; they do not produce generated `remoteadb` API commands.

Unversioned core routes are emitted as the `legacy` generation. Versioned core
and service routes are grouped by `v0`, `v1`, or `v2`; independent API families
such as `pipelines-v0`, `authn2`, `authz2`, and `foundry` retain distinct
generation names. This partition feeds the locked generation-collision rule.
