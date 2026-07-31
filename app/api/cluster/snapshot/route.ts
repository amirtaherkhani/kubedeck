import { NextResponse } from "next/server"

import { getCurrentAdmin } from "@/lib/auth"
import {
  AgentConfigurationError,
  requestClusterAgent,
} from "@/lib/kubedeck-agent"

export const dynamic = "force-dynamic"

export async function GET(request: Request) {
  if (!(await getCurrentAdmin())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  try {
    const upstream = await requestClusterAgent("/v1/snapshot", {
      headers: { Accept: "application/json" },
      signal: request.signal,
    })
    if (!upstream.ok) {
      return agentUnavailable(upstream.status)
    }
    return new Response(upstream.body, {
      status: httpStatus(upstream.status),
      headers: {
        "Content-Type":
          upstream.headers.get("Content-Type") ??
          "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      },
    })
  } catch (error) {
    return agentError(error)
  }
}

function agentError(error: unknown) {
  if (!(error instanceof AgentConfigurationError)) {
    console.error("[KubeDeck] Cluster agent snapshot request failed.", error)
  }
  return NextResponse.json(
    {
      error:
        error instanceof AgentConfigurationError
          ? "Cluster agent is not configured."
          : "Cluster agent is unavailable.",
    },
    { status: 503 }
  )
}

function agentUnavailable(status: number) {
  return NextResponse.json(
    {
      error: "Cluster agent rejected the snapshot request.",
      upstreamStatus: status,
    },
    { status: 503 }
  )
}

function httpStatus(status: number) {
  return status >= 200 && status <= 599 ? status : 200
}
