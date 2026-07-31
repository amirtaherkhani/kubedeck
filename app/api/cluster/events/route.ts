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

  const headers = new Headers({ Accept: "text/event-stream" })
  const lastEventID = request.headers.get("Last-Event-ID")
  if (lastEventID) headers.set("Last-Event-ID", lastEventID)

  try {
    const upstream = await requestClusterAgent("/v1/events", {
      headers,
      signal: request.signal,
    })
    if (!upstream.ok || !upstream.body) {
      return NextResponse.json(
        {
          error: "Cluster agent rejected the event stream.",
          upstreamStatus: upstream.status,
        },
        { status: 503 }
      )
    }
    return new Response(upstream.body, {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache, no-store",
        Connection: "keep-alive",
        "X-Accel-Buffering": "no",
        "Content-Encoding": "identity",
      },
    })
  } catch (error) {
    if (!(error instanceof AgentConfigurationError)) {
      console.error("[KubeDeck] Cluster agent SSE request failed.", error)
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
