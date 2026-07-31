import { env as cloudflareEnv } from "cloudflare:workers"

import {
  parseClusterSnapshot,
  type ClusterSnapshot,
} from "@/lib/kubedeck-cluster"

const AGENT_URL_KEY = "KUBEDECK_AGENT_URL"
const AGENT_TOKEN_KEY = "KUBEDECK_AGENT_TOKEN"

export class AgentConfigurationError extends Error {}

export function requestClusterAgent(
  pathname: "/v1/snapshot" | "/v1/events" | "/v1/dns/config",
  init: RequestInit = {}
) {
  const environment = cloudflareEnv as unknown as Record<string, unknown>
  const configuredURL =
    typeof environment[AGENT_URL_KEY] === "string"
      ? environment[AGENT_URL_KEY].trim()
      : ""
  if (!configuredURL) {
    throw new AgentConfigurationError(
      "KUBEDECK_AGENT_URL is not configured."
    )
  }

  const baseURL = new URL(configuredURL)
  if (baseURL.protocol !== "http:" && baseURL.protocol !== "https:") {
    throw new AgentConfigurationError(
      "KUBEDECK_AGENT_URL must use HTTP or HTTPS."
    )
  }
  baseURL.pathname = `${baseURL.pathname.replace(/\/+$/u, "")}${pathname}`
  baseURL.search = ""
  baseURL.hash = ""

  const headers = new Headers(init.headers)
  const token =
    typeof environment[AGENT_TOKEN_KEY] === "string"
      ? environment[AGENT_TOKEN_KEY].trim()
      : ""
  if (token) headers.set("Authorization", `Bearer ${token}`)

  return fetch(baseURL, {
    ...init,
    headers,
    cache: "no-store",
  })
}

export async function readClusterSnapshot(): Promise<ClusterSnapshot> {
  const response = await requestClusterAgent("/v1/snapshot", {
    headers: { Accept: "application/json" },
    signal: AbortSignal.timeout(3_000),
  })
  if (!response.ok) {
    throw new Error(`Cluster agent returned HTTP ${response.status}.`)
  }
  return parseClusterSnapshot(await response.json())
}

export async function readInitialClusterSnapshot() {
  try {
    return await readClusterSnapshot()
  } catch (error) {
    if (!(error instanceof AgentConfigurationError)) {
      console.error("[KubeDeck] Initial cluster snapshot failed.", error)
    }
    return null
  }
}
