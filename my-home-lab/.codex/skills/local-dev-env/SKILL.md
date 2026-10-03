---
name: local-dev-env
description: Read or change local-development environment values through the project-aware local-dev-env-mcp Infisical connector. Use whenever an agent needs env configuration, credentials, API keys, connection settings, or Kubernetes runtime values for a home-lab or registered local project.
---

# Local development environment values

Use the `local-dev-env` MCP tools. Project, environment, and Infisical path are selected automatically from the current workspace. Never ask the user for an Infisical password, project ID, profile, token, or secret path.

1. Call `env_context` only when the selected project or path is uncertain.
2. Call `env_list` to discover names without exposing values.
3. Call `env_get` only for values needed by the current task.
4. Use `env_set` or `env_set_many` for requested configuration changes. Keys must be uppercase environment names.
5. Use `env_delete` only when the user explicitly requested deletion.

For a chart with several registered service folders, pass the required descendant `path` to the data tool. Never pass a path outside the base returned by `env_context`.

Do not place returned values in chat, logs, files, Git, shell arguments, or test output. For a command that needs multiple values, prefer `infisical run` through the existing project helper rather than exporting a dotenv file.

Kubernetes does not use this MCP server. Charts must use the Infisical Secrets Operator with `includeAllSecrets: true` to reconcile every value in a service folder into a service-scoped native Secret. Consume that Secret with `envFrom.secretRef` when the chart supports whole-Secret environment injection; otherwise use the chart's native `existingSecret` or `secretKeyRef` interface. Add `secrets.infisical.com/auto-reload: "true"` to the workload controller so synchronized changes restart pods. Never inject one service's Secret into a different workload.
