#!/usr/bin/env node

import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const [input, outputDir] = process.argv.slice(2);
if (!input || !outputDir) {
  console.error("usage: main.mjs <bundled-openapi.json> <output-directory>");
  process.exit(2);
}

const source = JSON.parse(await readFile(input, "utf8"));
const sourcePathCount = Object.keys(source.paths).length;
const excludedPaths = new Set([
  "/sys/health",
  "/device/v0/devices/",
]);

const uuidParameter = (name) => ({
  name,
  in: "path",
  required: true,
  schema: { type: "string", format: "uuid" },
});

const jsonResponse = (description, schema = { type: "object", additionalProperties: true }) => ({
  description,
  content: { "application/json": { schema } },
});

const errorResponses = {
  "400": jsonResponse("Bad request"),
  "401": jsonResponse("Unauthorized"),
  "404": jsonResponse("Not found"),
};

const remoteADBProperties = {
  id: { type: "string", format: "uuid" },
  enterprise: { type: "string", format: "uuid" },
  device: { type: "string", format: "uuid" },
  device_name: { type: "string" },
  ip: { type: ["string", "null"] },
  client_port: { type: ["string", "null"] },
  device_port: { type: ["string", "null"] },
  remoteadb_host: { type: ["string", "null"] },
  client_certificate: { type: ["string", "null"] },
  device_certificate: { type: ["string", "null"] },
  state: { type: "string" },
  errors: { type: ["string", "null"] },
};
const remoteADBResponse = { type: "object", properties: remoteADBProperties };
const remoteADBParameters = [uuidParameter("enterprise_id"), uuidParameter("device_id")];
const remoteADBSource = [
  "src/services/shoonyacloud/shoonyapoc/api/remoteadb/urls.py:13-22",
  "src/services/shoonyacloud/shoonyapoc/api/remoteadb/views/remoteadb_connection.py:9-20",
];

// Platform source: api/remoteadb/urls.py:13-22 registers a ModelViewSet below enterprise/device.
source.paths["/v0/enterprise/{enterprise_id}/device/{device_id}/remoteadb/"] = {
  get: {
    operationId: "listRemoteADBConnections",
    summary: "List remote ADB connections for a device",
    tags: ["Remote ADB"],
    parameters: [
      ...remoteADBParameters,
      { name: "limit", in: "query", schema: { type: "integer", minimum: 1 } },
      { name: "offset", in: "query", schema: { type: "integer", minimum: 0 } },
    ],
    responses: {
      "200": jsonResponse("Remote ADB connections", {
        type: "object",
        properties: {
          count: { type: "integer" },
          next: { type: ["string", "null"], format: "uri" },
          previous: { type: ["string", "null"], format: "uri" },
          results: { type: "array", items: remoteADBResponse },
        },
      }),
      ...errorResponses,
    },
    "x-esper-platform-source": remoteADBSource,
  },
  post: {
    operationId: "createRemoteADBConnection",
    summary: "Start a remote ADB connection",
    tags: ["Remote ADB"],
    parameters: remoteADBParameters,
    requestBody: {
      required: true,
      content: {
        "application/json": {
          schema: {
            type: "object",
            required: ["client_certificate"],
            properties: { client_certificate: { type: "string" } },
          },
        },
      },
    },
    responses: { "201": jsonResponse("Remote ADB connection created", remoteADBResponse), ...errorResponses },
    "x-esper-platform-source": remoteADBSource,
  },
};

source.paths["/v0/enterprise/{enterprise_id}/device/{device_id}/remoteadb/{remoteadb_id}/"] = {
  get: {
    operationId: "getRemoteADBConnection",
    summary: "Get remote ADB connection status",
    tags: ["Remote ADB"],
    parameters: [...remoteADBParameters, uuidParameter("remoteadb_id")],
    responses: { "200": jsonResponse("Remote ADB connection", remoteADBResponse), ...errorResponses },
    "x-esper-platform-source": remoteADBSource,
  },
  delete: {
    operationId: "deleteRemoteADBConnection",
    summary: "Delete a remote ADB connection",
    tags: ["Remote ADB"],
    parameters: [...remoteADBParameters, uuidParameter("remoteadb_id")],
    responses: { "204": { description: "Remote ADB connection deleted" }, ...errorResponses },
    "x-esper-platform-source": remoteADBSource,
  },
};

// Platform source: shoonyapoc/urls.py:158-160 and telemetry_graph.py:44-74.
source.paths["/graph/{category}/{metric}/"] = {
  get: {
    operationId: "getTelemetryGraphData",
    summary: "Get telemetry graph data",
    tags: ["Device Telemetry"],
    parameters: [
      { name: "category", in: "path", required: true, schema: { type: "string" } },
      { name: "metric", in: "path", required: true, schema: { type: "string" } },
      { name: "from_time", in: "query", required: true, schema: { type: "string", format: "date-time" } },
      { name: "to_time", in: "query", required: true, schema: { type: "string", format: "date-time" } },
      { name: "period", in: "query", required: true, schema: { type: "string" } },
      { name: "statistic", in: "query", required: true, schema: { type: "string" } },
      uuidParameter("device_id"),
      { name: "enterprise_id", in: "query", schema: { type: "string", format: "uuid" } },
    ].map((parameter) => parameter.name === "device_id" ? { ...parameter, in: "query", required: true } : parameter),
    responses: { "200": jsonResponse("Telemetry data"), ...errorResponses },
    "x-esper-platform-source": [
      "src/services/shoonyacloud/shoonyapoc/shoonyapoc/urls.py:158-160",
      "src/services/shoonyacloud/shoonyapoc/api/device/views/telemetry_graph.py:44-74",
    ],
  },
};

const groupCommandParameters = [uuidParameter("enterprise_id"), uuidParameter("group_id")];
const groupCommandSource = [
  "src/services/shoonyacloud/shoonyapoc/api/enterprise/urls.py:250-256",
  "src/services/shoonyacloud/shoonyapoc/api/device/views/group_command.py:43-93",
];
const groupCommandResponse = {
  type: "object",
  properties: {
    id: { type: "string", format: "uuid" },
    command: { type: "string" },
    command_args: { type: "object", additionalProperties: true },
    state: { type: "string" },
    details: { type: "object", additionalProperties: true },
  },
};

// Platform source: api/enterprise/urls.py:250-256 nests GroupCommandViewSet under devicegroup.
source.paths["/enterprise/{enterprise_id}/devicegroup/{group_id}/command/"] = {
  get: {
    operationId: "listLegacyDeviceGroupCommands",
    summary: "List commands for a device group",
    tags: ["Device Group Command"],
    parameters: [
      ...groupCommandParameters,
      { name: "limit", in: "query", schema: { type: "integer", minimum: 1 } },
      { name: "offset", in: "query", schema: { type: "integer", minimum: 0 } },
    ],
    responses: {
      "200": jsonResponse("Device group commands", {
        type: "object",
        properties: {
          count: { type: "integer" },
          next: { type: ["string", "null"], format: "uri" },
          previous: { type: ["string", "null"], format: "uri" },
          results: { type: "array", items: groupCommandResponse },
        },
      }),
      ...errorResponses,
    },
    "x-esper-platform-source": groupCommandSource,
  },
  post: {
    operationId: "createLegacyDeviceGroupCommand",
    summary: "Send a command to a device group",
    tags: ["Device Group Command"],
    parameters: groupCommandParameters,
    requestBody: {
      required: true,
      content: {
        "application/json": {
          schema: {
            type: "object",
            required: ["command"],
            properties: {
              command: { type: "string" },
              command_args: { type: "object", additionalProperties: true },
              schedule: { type: "object", additionalProperties: true },
            },
          },
        },
      },
    },
    responses: { "201": jsonResponse("Device group command created", groupCommandResponse), ...errorResponses },
    "x-esper-platform-source": groupCommandSource,
  },
};

source.paths["/enterprise/{enterprise_id}/devicegroup/{group_id}/command/{command_id}/"] = {
  get: {
    operationId: "getLegacyDeviceGroupCommand",
    summary: "Get a device group command",
    tags: ["Device Group Command"],
    parameters: [...groupCommandParameters, uuidParameter("command_id")],
    responses: { "200": jsonResponse("Device group command", groupCommandResponse), ...errorResponses },
    "x-esper-platform-source": groupCommandSource,
  },
};

function generationFor(apiPath) {
  if (apiPath.startsWith("/authn2/")) return "authn2";
  if (apiPath.startsWith("/authz2/")) return "authz2";
  if (apiPath.startsWith("/pipelines/v0/")) return "pipelines-v0";
  if (apiPath.startsWith("/v1/foundry/")) return "foundry";
  if (/^\/(apps|commands|device|onboarding|report|tenant)\/v0\//.test(apiPath)) return "v0";
  if (apiPath.startsWith("/v1/")) return "v1";
  if (apiPath.startsWith("/v2/") || apiPath.startsWith("/api/v2/")) return "v2";
  if (apiPath.startsWith("/v0/")) return "v0";
  if (apiPath.startsWith("/enterprise/") || apiPath.startsWith("/user") ||
      apiPath.startsWith("/graph/")) return "legacy";
  throw new Error(`no generation mapping for ${apiPath}`);
}

const methods = new Set(["get", "post", "put", "patch", "delete"]);
const resourceNames = new Map([
  ["devicegroup", "device-group"],
  ["devicegroups", "device-group"],
  ["operationlists", "operation-list"],
  ["pipelines", "pipeline"],
  ["runs", "pipeline-run"],
  ["stageruns", "stage-run"],
  ["stages", "stage"],
  ["targetlists", "target-list"],
  ["targetruns", "target-run"],
  ["targets", "target"],
]);

function words(value) {
  const normalized = value
    .replaceAll("APNs", "Apns")
    .replaceAll("APNS", "Apns")
    .replaceAll("EMM", "Emm")
    .replaceAll("VPP", "Vpp")
    .replaceAll("DEP", "Dep")
    .replaceAll("OTA", "Ota")
    .replaceAll("ADB", "Adb");
  return normalized
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .replace(/[^A-Za-z0-9]+/g, " ")
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean);
}

function singular(word) {
  const irregular = new Map([
    ["apps", "app"],
    ["policies", "policy"],
    ["activities", "activity"],
    ["companies", "company"],
    ["statuses", "status"],
    ["data", "data"],
    ["apns", "apns"],
  ]);
  if (irregular.has(word)) return irregular.get(word);
  if (word.endsWith("ies") && word.length > 4) return `${word.slice(0, -3)}y`;
  if (word.endsWith("sses") || word.endsWith("uses")) return word.slice(0, -2);
  if (word.endsWith("s") && !word.endsWith("ss") && word !== "status") return word.slice(0, -1);
  return word;
}

function nounFromTag(operation) {
  const tag = operation.tags?.[0] ?? "operation";
  const parts = words(tag)
    .filter((part) => !/^\d+$/.test(part))
    .filter((part) => !["api", "deprecated", "v0", "v1", "v2", "v3"].includes(part));
  if (parts.length === 0) return "operation";
  for (let index = 0; index < parts.length; index++) parts[index] = singular(parts[index]);
  return parts.join("-");
}

const actionPrefixes = [
  "partialupdate", "partial", "bulk", "list", "get", "create", "post", "update",
  "put", "patch", "edit", "delete", "upload", "download", "generate", "renew",
  "restore", "apply", "remove", "add",
];

function nounFromPath(apiPath, operation) {
  const ignored = new Set([
    "api", "v0", "v1", "v2", "v3", "authn2", "authz2", "pipelines", "enterprise",
    "download", "upload",
  ]);
  const parts = apiPath.split("/")
    .filter(Boolean)
    .filter((part) => !part.startsWith("{"))
    .flatMap(words)
    .filter((part) => !ignored.has(part));
  if (parts.length === 0) return nounFromTag(operation);
  for (let index = 0; index < parts.length; index++) parts[index] = singular(parts[index]);
  return parts.join("-");
}

function relationshipCollectionNoun(apiPath, generation) {
  const segments = apiPath.split("/").filter(Boolean);
  if (segments.length < 2) return "";
  const finalSegment = segments.at(-1);
  if (finalSegment.startsWith("{") && generation !== "pipelines-v0") return "";
  const collection = finalSegment.startsWith("{") ? segments.at(-2) : finalSegment;
  const parentIndex = finalSegment.startsWith("{") ? segments.length - 3 : segments.length - 2;
  const parent = segments[parentIndex];
  if (!parent?.startsWith("{") || !parent.endsWith("}")) return "";
  if (resourceNames.has(collection.toLowerCase())) return resourceNames.get(collection.toLowerCase());
  const parts = words(collection);
  if (parts.length === 0) return "";
  for (let index = 0; index < parts.length; index++) parts[index] = singular(parts[index]);
  return parts.join("-");
}

function nounFor(apiPath, operation, generation) {
  const relationshipNoun = relationshipCollectionNoun(apiPath, generation);
  if (relationshipNoun) return relationshipNoun;
  if (!operation.operationId) return nounFromPath(apiPath, operation);
  let parts = words(operation.operationId);
  if (parts[0] === "partial" && parts[1] === "update") parts = parts.slice(2);
  else if (parts[0] === "bulk" && ["add", "create", "update", "delete"].includes(parts[1])) parts = parts.slice(2);
  else if (actionPrefixes.includes(parts[0])) parts = parts.slice(1);
  if (parts[0] === "all") parts = parts.slice(1);
  parts = parts.filter((part) => !/^v\d+$/.test(part)).map((part) => part.replace(/v\d+$/, ""));
  const by = parts.indexOf("by");
  if (by > 0) parts = parts.slice(0, by);
  while (["url", "id"].includes(parts.at(-1))) parts.pop();
  if (parts.length === 0) return nounFromTag(operation);
  for (let index = 0; index < parts.length; index++) parts[index] = singular(parts[index]);
  return parts.join("-");
}

function operationHasPagination(operation) {
  const names = new Set((operation.parameters ?? []).map((parameter) => parameter.name));
  return names.has("limit") && names.has("offset");
}

function isCollectionPath(apiPath) {
  const finalSegment = apiPath.replace(/\/$/, "").split("/").at(-1) ?? "";
  return !finalSegment.startsWith("{") && finalSegment.endsWith("s") &&
    !["status", "metrics"].includes(finalSegment);
}

function verbFor(method, apiPath, operation, collectionPaths) {
  const id = operation.operationId ?? "";
  const parts = words(id);
  const first = parts[0] ?? "";
  if (first === "list" || (first === "get" && parts[1] === "all")) return "list";
  if (first === "get") return operationHasPagination(operation) || isCollectionPath(apiPath) ||
    collectionPaths.has(apiPath) ? "list" : "get";
  if (["create", "post"].includes(first)) return "create";
  if (first === "partialupdate" || (first === "partial" && parts[1] === "update")) return "partial-update";
  if (first === "patch") return "patch";
  if (["update", "put", "edit"].includes(first)) return "update";
  if (first === "delete") return "delete";
  if (first === "bulk" && parts[1]) return `bulk-${parts[1]}`;
  if (["upload", "download", "generate", "renew", "restore", "apply", "remove", "add"].includes(first)) return first;

  if (method === "get") {
    const finalSegment = apiPath.replace(/\/$/, "").split("/").at(-1);
    return collectionPaths.has(apiPath) || operationHasPagination(operation) || isCollectionPath(apiPath)
      ? "list"
      : "get";
  }
  if (method === "post") {
    const finalSegment = apiPath.replace(/\/$/, "").split("/").at(-1) ?? "";
    if (["upload", "download", "generate", "renew", "restore", "apply", "remove", "run", "execute"].includes(finalSegment)) {
      return finalSegment;
    }
    return "create";
  }
  if (method === "put") return "update";
  if (method === "patch") return "patch";
  return "delete";
}

function paginationFor(apiPath, operation, verb) {
  if (verb !== "list" || !operationHasPagination(operation)) return "none";
  if (apiPath === "/v2/devices/" || /^\/(apps\/v0|v2\/(itunesapps|tenant-apps|appleappstore|apps|webclips|provisioning-profiles|preferred-regions|esper-apps|tenant-esper-apps))/.test(apiPath)) {
    return "apps-envelope";
  }
  return "limit-offset";
}

function serviceFamily(apiPath) {
  return apiPath.split("/").filter(Boolean).slice(0, 2).join("/");
}

function requiredOneOfFor(apiPath) {
  if (apiPath === "/commands/v0/status/") return ["request", "device"];
  if (apiPath === "/v2/itunesapps") return ["app_id", "apple_app_id"];
  return [];
}

function scopeParentFor(apiPath, verb, generation) {
  const segments = apiPath.split("/").filter(Boolean);
  if (generation !== "pipelines-v0") {
    if (!["list", "create", "add"].includes(verb) || segments.length === 0 || segments.at(-1).startsWith("{")) return "";
    for (let index = segments.length - 2; index >= 0; index--) {
      const segment = segments[index];
      if (!segment.startsWith("{") || !segment.endsWith("}")) continue;
      const parentSegment = segments[index - 1];
      if (parentSegment && !parentSegment.startsWith("{")) {
        const parentParts = words(parentSegment);
        if (parentParts.length > 0) {
          for (let part = 0; part < parentParts.length; part++) parentParts[part] = singular(parentParts[part]);
          return parentParts.join("-");
        }
      }
      const parameter = segment.slice(1, -1).replace(/_id$/i, "").replace(/Id$/, "");
      const parts = words(parameter);
      if (parts.length === 0) return "";
      parts[parts.length - 1] = singular(parts[parts.length - 1]);
      return parts.join("-");
    }
    return "";
  }
  const finalIsID = segments.at(-1)?.startsWith("{");
  const parentIDIndex = finalIsID ? segments.length - 3 : segments.length - 2;
  const parentResource = segments[parentIDIndex - 1];
  if (parentIDIndex < 0 || !segments[parentIDIndex]?.startsWith("{") || !parentResource || parentResource.startsWith("{")) return "";
  if (resourceNames.has(parentResource.toLowerCase())) return resourceNames.get(parentResource.toLowerCase());
  const parts = words(parentResource);
  if (parts.length === 0) return "";
  for (let index = 0; index < parts.length; index++) parts[index] = singular(parts[index]);
  return parts.join("-");
}

function annotateOperations(spec, generation) {
  const collectionPaths = new Set();
  const paths = Object.keys(spec.paths);
  for (const apiPath of paths) {
    const prefix = apiPath.endsWith("/") ? apiPath : `${apiPath}/`;
    if (paths.some((candidate) => candidate !== apiPath && candidate.startsWith(`${prefix}{`))) {
      collectionPaths.add(apiPath);
    }
  }
  let operationCount = 0;
  const commands = new Map();
  const appsEnvelopeFamilies = new Set();
  for (const [apiPath, pathItem] of Object.entries(spec.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!methods.has(method)) continue;
      const verb = verbFor(method, apiPath, operation, collectionPaths);
      if (paginationFor(apiPath, operation, verb) === "apps-envelope") {
        appsEnvelopeFamilies.add(serviceFamily(apiPath));
      }
    }
  }
  for (const [apiPath, pathItem] of Object.entries(spec.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!methods.has(method)) continue;
      operationCount++;
	   const noun = nounFor(apiPath, operation, generation);
      const verb = verbFor(method, apiPath, operation, collectionPaths);
      const identity = `${noun} ${verb}`;
	   const scopeParent = scopeParentFor(apiPath, verb, generation);
      operation["x-esper-destructive"] = method === "delete" ||
        /\b(delete|remove|wipe|factory|revoke|terminate|uninstall|unapply)\b/i.test(`${operation.operationId ?? ""} ${operation.summary ?? ""}`);
      operation["x-esper-pagination"] = paginationFor(apiPath, operation, verb);
      const requiredOneOf = requiredOneOfFor(apiPath);
      if (requiredOneOf.length > 0) operation["x-esper-require-one-of"] = requiredOneOf;
      if (apiPath === "/v2/subgroups") {
        const parentGroups = (operation.parameters ?? []).find((parameter) => parameter.name === "parent_group_ids");
        if (parentGroups) parentGroups.required = true;
      }
      if (appsEnvelopeFamilies.has(serviceFamily(apiPath))) {
        operation["x-esper-response-envelope"] = "apps-envelope";
      }
      operation["x-esper-verb"] = verb;
      operation["x-esper-noun"] = noun;
      if (!operation.summary) operation.summary = `${verb.charAt(0).toUpperCase()}${verb.slice(1)} ${noun}`;
      if (scopeParent) operation["x-esper-scope-parent"] = scopeParent;
      if (!commands.has(identity)) commands.set(identity, []);
      commands.get(identity).push({ apiPath, method, scopeParent });
    }
  }

  const collisions = [];
  for (const [identity, operations] of commands) {
    if (operations.length < 2) continue;
    const unscoped = operations.filter((operation) => !operation.scopeParent);
    const scoped = operations.filter((operation) => operation.scopeParent);
    const uniqueScopes = new Set(scoped.map((operation) => operation.scopeParent));
    const explainedByScope = scoped.length > 0 && unscoped.length <= 1 &&
      uniqueScopes.size === scoped.length;
    if (!explainedByScope) collisions.push({ generation, identity, operations });
  }
  return { operationCount, collisions };
}

function decodePointerPart(part) {
  return part.replaceAll("~1", "/").replaceAll("~0", "~");
}

function referencedComponents(value, found = new Set()) {
  if (Array.isArray(value)) {
    for (const item of value) referencedComponents(item, found);
    return found;
  }
  if (!value || typeof value !== "object") return found;

  if (typeof value.$ref === "string" && value.$ref.startsWith("#/components/")) {
    found.add(value.$ref);
  }
  for (const child of Object.values(value)) referencedComponents(child, found);
  return found;
}

function componentAt(ref) {
  const parts = ref.slice(2).split("/").map(decodePointerPart);
  let value = source;
  for (const part of parts) value = value?.[part];
  if (value === undefined) throw new Error(`unresolved component reference ${ref}`);
  return { parts, value };
}

function componentsFor(paths) {
  const pending = [...referencedComponents(paths)];
  const visited = new Set();
  const components = {};

  while (pending.length > 0) {
    const ref = pending.pop();
    if (visited.has(ref)) continue;
    visited.add(ref);

    const { parts, value } = componentAt(ref);
    const [, type, name] = parts;
    components[type] ??= {};
    components[type][name] = value;
    for (const nested of referencedComponents(value)) pending.push(nested);
  }

  const securityNames = new Set();
  function collectSecurity(value) {
    if (Array.isArray(value)) {
      for (const item of value) collectSecurity(item);
      return;
    }
    if (!value || typeof value !== "object") return;
    if (Array.isArray(value.security)) {
      for (const requirement of value.security) {
        for (const name of Object.keys(requirement)) securityNames.add(name);
      }
    }
    for (const child of Object.values(value)) collectSecurity(child);
  }
  collectSecurity({ security: source.security, paths });
  for (const name of securityNames) {
    const scheme = source.components?.securitySchemes?.[name];
    if (!scheme) throw new Error(`undefined security scheme ${name}`);
    components.securitySchemes ??= {};
    components.securitySchemes[name] = scheme;
  }

  return components;
}

function convertSchemasTo31(value) {
  if (Array.isArray(value)) {
    for (const item of value) convertSchemasTo31(item);
    return;
  }
  if (!value || typeof value !== "object") return;

  if (value.nullable === true) {
    if (typeof value.type === "string") value.type = [value.type, "null"];
    else if (Array.isArray(value.type) && !value.type.includes("null")) value.type.push("null");
    else if (value.$ref) {
      value.anyOf = [{ $ref: value.$ref }, { type: "null" }];
      delete value.$ref;
    }
  }
  delete value.nullable;

  if (value.exclusiveMinimum === true && typeof value.minimum === "number") {
    value.exclusiveMinimum = value.minimum;
    delete value.minimum;
  } else if (value.exclusiveMinimum === false) {
    delete value.exclusiveMinimum;
  }
  if (value.exclusiveMaximum === true && typeof value.maximum === "number") {
    value.exclusiveMaximum = value.maximum;
    delete value.maximum;
  } else if (value.exclusiveMaximum === false) {
    delete value.exclusiveMaximum;
  }

  for (const child of Object.values(value)) convertSchemasTo31(child);
}

const byGeneration = new Map();
for (const [sourcePath, pathItem] of Object.entries(source.paths)) {
  const apiPath = sourcePath === "/api/v2/subgroups/" ? "/v2/subgroups" : sourcePath;
  if (excludedPaths.has(apiPath)) continue;
  const generation = generationFor(apiPath);
  if (!byGeneration.has(generation)) byGeneration.set(generation, {});
  byGeneration.get(generation)[apiPath] = pathItem;
}

await mkdir(outputDir, { recursive: true });
const manifest = {
  sourcePathCount,
  addedPathCount: Object.keys(source.paths).length - sourcePathCount,
  excludedPaths: [...excludedPaths].sort(),
  emittedPathCount: 0,
  operationCount: 0,
  generations: {},
};
const unexplainedCollisions = [];

for (const generation of [...byGeneration.keys()].sort()) {
  const paths = byGeneration.get(generation);
  const usedTags = new Set();
  for (const pathItem of Object.values(paths)) {
    for (const operation of Object.values(pathItem)) {
      if (operation && typeof operation === "object" && Array.isArray(operation.tags)) {
        for (const tag of operation.tags) usedTags.add(tag);
      }
    }
  }

  const spec = {
    openapi: "3.1.0",
    info: {
      title: `${source.info.title} - ${generation}`,
      version: source.info.version,
      "x-esper-generation": generation,
      "x-esper-auth": "bearer",
    },
    servers: source.servers,
    tags: (source.tags ?? []).filter((tag) => usedTags.has(tag.name)),
    paths,
    components: componentsFor(paths),
  };
  if (source.security) spec.security = source.security;
  const annotation = annotateOperations(spec, generation);
  manifest.operationCount += annotation.operationCount;
  unexplainedCollisions.push(...annotation.collisions);
  convertSchemasTo31(spec);

  const filename = `${generation}.yaml`;
  await writeFile(path.join(outputDir, filename), `${JSON.stringify(spec, null, 2)}\n`);
  manifest.generations[generation] = Object.keys(paths).length;
  manifest.emittedPathCount += Object.keys(paths).length;
}

if (unexplainedCollisions.length > 0) {
  const details = unexplainedCollisions.map((collision) => {
    const operations = collision.operations
      .map((operation) => `${operation.method.toUpperCase()} ${operation.apiPath}`)
      .join(", ");
    return `${collision.generation}: ${collision.identity}: ${operations}`;
  });
  throw new Error(`unexplained same-generation command collisions:\n${details.join("\n")}`);
}

await writeFile(
  path.join(outputDir, "manifest.json"),
  `${JSON.stringify(manifest, null, 2)}\n`,
);

console.log(`emitted ${manifest.emittedPathCount} paths across ${byGeneration.size} generations`);
