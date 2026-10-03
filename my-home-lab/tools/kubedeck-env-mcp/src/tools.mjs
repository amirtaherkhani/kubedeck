import { posix } from "node:path";
import { contextSummary } from "./context.mjs";

const KEY_PATTERN = "^[A-Z][A-Z0-9_]*$";
const keyExpression = new RegExp(KEY_PATTERN);
const pathProperty = { type: "string", pattern: "^/" };

export const toolDefinitions = [
  {
    name: "env_context",
    description: "Show the automatically selected local Infisical project context. No secret values are returned.",
    inputSchema: { type: "object", additionalProperties: false }
  },
  {
    name: "env_list",
    description: "List secret names in the selected project's local development scope. Values are not returned.",
    inputSchema: { type: "object", properties: { path: pathProperty }, additionalProperties: false }
  },
  {
    name: "env_get",
    description: "Read one environment value from the selected local development project.",
    inputSchema: {
      type: "object",
      properties: { name: { type: "string", pattern: KEY_PATTERN }, path: pathProperty },
      required: ["name"],
      additionalProperties: false
    }
  },
  {
    name: "env_set",
    description: "Create or update one environment value in the selected local development project.",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", pattern: KEY_PATTERN },
        value: { type: "string" },
        path: pathProperty
      },
      required: ["name", "value"],
      additionalProperties: false
    }
  },
  {
    name: "env_set_many",
    description: "Create or update multiple environment values in the selected local development project.",
    inputSchema: {
      type: "object",
      properties: {
        values: {
          type: "object",
          propertyNames: { pattern: KEY_PATTERN },
          additionalProperties: { type: "string" },
          minProperties: 1
        },
        path: pathProperty
      },
      required: ["values"],
      additionalProperties: false
    }
  },
  {
    name: "env_delete",
    description: "Delete one shared environment value from the selected local development project. Use only when deletion was explicitly requested.",
    inputSchema: {
      type: "object",
      properties: { name: { type: "string", pattern: KEY_PATTERN }, path: pathProperty },
      required: ["name"],
      additionalProperties: false
    }
  }
];

function result(data) {
  return { content: [{ type: "text", text: JSON.stringify(data) }], structuredContent: data };
}

function requireName(name) {
  if (typeof name !== "string" || !keyExpression.test(name)) {
    throw new Error("name must be an uppercase environment key");
  }
}

function requireOnlyKeys(value, allowed) {
  if (!value || typeof value !== "object" || Array.isArray(value) ||
    Object.keys(value).some((key) => !allowed.includes(key))) {
    throw new Error(`arguments may contain only: ${allowed.join(", ") || "no fields"}`);
  }
}

function resolvePath(context, requested) {
  if (requested === undefined) return context.path;
  if (typeof requested !== "string" || !requested.startsWith("/") || posix.normalize(requested) !== requested) {
    throw new Error("path must be a normalized absolute Infisical path");
  }
  if (context.path !== "/" && requested !== context.path && !requested.startsWith(`${context.path}/`)) {
    throw new Error(`path must stay within ${context.path}`);
  }
  return requested;
}

export async function callTool({ name, arguments: args = {} }, { context, store }) {
  switch (name) {
    case "env_context":
      requireOnlyKeys(args, []);
      return result(contextSummary(context));
    case "env_list": {
      requireOnlyKeys(args, ["path"]);
      const path = resolvePath(context, args.path);
      return result({ path, names: await store.list(path) });
    }
    case "env_get": {
      requireOnlyKeys(args, ["name", "path"]);
      requireName(args.name);
      const path = resolvePath(context, args.path);
      const secret = await store.get(args.name, path);
      return result({ name: secret.secretKey, path, value: secret.secretValue });
    }
    case "env_set": {
      requireOnlyKeys(args, ["name", "value", "path"]);
      requireName(args.name);
      if (typeof args.value !== "string") throw new Error("value must be a string");
      const path = resolvePath(context, args.path);
      return result({ name: args.name, path, action: await store.set(args.name, args.value, path) });
    }
    case "env_set_many": {
      requireOnlyKeys(args, ["values", "path"]);
      if (!args.values || typeof args.values !== "object" || Array.isArray(args.values) || Object.keys(args.values).length === 0) {
        throw new Error("values must be a non-empty object");
      }
      for (const [key, value] of Object.entries(args.values)) {
        requireName(key);
        if (typeof value !== "string") throw new Error(`value for ${key} must be a string`);
      }
      const path = resolvePath(context, args.path);
      return result({ path, results: await store.setMany(args.values, path) });
    }
    case "env_delete": {
      requireOnlyKeys(args, ["name", "path"]);
      requireName(args.name);
      const path = resolvePath(context, args.path);
      await store.delete(args.name, path);
      return result({ name: args.name, path, action: "deleted" });
    }
    default:
      throw new Error(`unknown tool: ${name}`);
  }
}
