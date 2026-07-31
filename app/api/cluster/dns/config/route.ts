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

  return proxyDNSRequest({
    headers: { Accept: "application/json" },
    signal: request.signal,
  })
}

export async function PUT(request: Request) {
  if (!(await getCurrentAdmin())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  return proxyDNSRequest({
    method: "PUT",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: await request.text(),
    signal: request.signal,
  })
}

async function proxyDNSRequest(init: RequestInit) {
  try {
    const upstream = await requestClusterAgent("/v1/dns/config", init)
    return new Response(upstream.body, {
      status: validStatus(upstream.status),
      headers: {
        "Content-Type":
          upstream.headers.get("Content-Type") ??
          "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      },
    })
  } catch (error) {
    if (!(error instanceof AgentConfigurationError)) {
      console.error("[KubeDeck] CoreDNS configuration request failed.", error)
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
}

function validStatus(status: number) {
  return status >= 200 && status <= 599 ? status : 502
}
