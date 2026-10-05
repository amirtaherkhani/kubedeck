import { NextResponse } from "next/server"
import { getCurrentAdmin } from "@/lib/auth"
import { AgentConfigurationError, requestClusterAgent } from "@/lib/kubedeck-agent"

export const dynamic = "force-dynamic"

type Context = { params: Promise<{ path: string[] }> }
const segment = /^[a-z0-9][a-z0-9.-]{0,252}$/u

async function proxy(request: Request, context: Context) {
  if (!(await getCurrentAdmin())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }
  const { path } = await context.params
  if (path.length === 0 || path.length > 5 || path.some((part) => !segment.test(part))) {
    return NextResponse.json({ error: "Invalid management path" }, { status: 400 })
  }
  const requestURL = new URL(request.url)
  const confirmation = request.headers.get("X-KubeDeck-Confirm")
  if (path[0] === "resources" && request.method === "POST" && !confirmation) {
    return NextResponse.json({ error: "Resource confirmation required" }, { status: 409 })
  }
  if (path[0] === "resources" && ["PUT", "PATCH", "DELETE"].includes(request.method)) {
    const namespace = requestURL.searchParams.get("namespace") ?? ""
    const name = requestURL.searchParams.get("name") ?? ""
    if (!name || confirmation !== `${namespace}/${name}`) {
      return NextResponse.json({ error: "Exact resource confirmation required" }, { status: 409 })
    }
  }
  if (path[0] === "workloads" && request.method === "POST" && path[4] !== "status") {
    if (confirmation !== `${path[1]}/${path[2]}/${path[3]}`) {
      return NextResponse.json({ error: "Exact workload confirmation required" }, { status: 409 })
    }
  }
  const agentPath = `/v1/manage/${path.join("/")}${requestURL.search}` as `/v1/manage/${string}`
  const headers = new Headers({ Accept: request.headers.get("Accept") ?? "application/json" })
  for (const name of ["Content-Type", "If-Match", "If-Match-UID", "X-KubeDeck-Confirm", "X-Request-ID"]) {
    const value = request.headers.get(name)
    if (value) headers.set(name, value)
  }
  const isWrite = !["GET", "HEAD"].includes(request.method)
  let body: ArrayBuffer | undefined
  if (isWrite) {
    if (Number(request.headers.get("Content-Length") ?? 0) > 1_048_576) {
      return NextResponse.json({ error: "Request too large" }, { status: 413 })
    }
    body = await request.arrayBuffer()
    if (body.byteLength > 1_048_576) {
      return NextResponse.json({ error: "Request too large" }, { status: 413 })
    }
  }
  try {
    const upstream = await requestClusterAgent(agentPath, { method: request.method, headers, body, signal: request.signal })
    return new Response(upstream.body, {
      status: upstream.status,
      headers: {
        "Content-Type": upstream.headers.get("Content-Type") ?? "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      },
    })
  } catch (error) {
    if (!(error instanceof AgentConfigurationError)) console.error("[KubeDeck] Agent management request failed.")
    return NextResponse.json({ error: "Cluster agent is unavailable." }, { status: 503 })
  }
}

export const GET = proxy
export const POST = proxy
export const PUT = proxy
export const PATCH = proxy
export const DELETE = proxy
