"use client"

import * as React from "react"

import {
  appendSnapshotHistory,
  parseClusterSnapshot,
  parseClusterStreamEnvelope,
  type ClusterSnapshot,
  type SnapshotHistory,
} from "@/lib/kubedeck-cluster"

export type ClusterConnectionStatus =
  | "connecting"
  | "live"
  | "reconnecting"
  | "unavailable"

export function useKubeDeckCluster(initialSnapshot: ClusterSnapshot | null) {
  const [snapshot, setSnapshot] = React.useState(initialSnapshot)
  const [history, setHistory] = React.useState<SnapshotHistory>(() =>
    initialSnapshot ? [initialSnapshot] : []
  )
  const [status, setStatus] = React.useState<ClusterConnectionStatus>(
    initialSnapshot ? "live" : "connecting"
  )
  const [error, setError] = React.useState("")
  const hasSnapshot = React.useRef(Boolean(initialSnapshot))

  React.useEffect(() => {
    let active = true

    function acceptSnapshot(next: ClusterSnapshot) {
      if (!active) return
      hasSnapshot.current = true
      setSnapshot(next)
      setHistory((current) => appendSnapshotHistory(current, next))
      setStatus("live")
      setError("")
    }

    void fetch("/api/cluster/snapshot", {
      headers: { Accept: "application/json" },
      cache: "no-store",
    })
      .then(async (response) => {
        if (!response.ok) throw new Error("Cluster agent snapshot unavailable.")
        acceptSnapshot(parseClusterSnapshot(await response.json()))
      })
      .catch((requestError: unknown) => {
        if (!active || hasSnapshot.current) return
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Cluster agent snapshot unavailable."
        )
      })

    const eventSource = new EventSource("/api/cluster/events")
    eventSource.onopen = () => {
      if (!active) return
      setStatus(hasSnapshot.current ? "live" : "connecting")
      setError("")
    }
    eventSource.addEventListener("snapshot", (event) => {
      try {
        const envelope = parseClusterStreamEnvelope(
          JSON.parse((event as MessageEvent<string>).data)
        )
        acceptSnapshot(parseClusterSnapshot(envelope.data))
      } catch (streamError) {
        if (!active) return
        setError(
          streamError instanceof Error
            ? streamError.message
            : "Invalid cluster event received."
        )
      }
    })
    eventSource.onerror = () => {
      if (!active) return
      setStatus(hasSnapshot.current ? "reconnecting" : "unavailable")
      setError("Live cluster stream disconnected; retrying automatically.")
    }

    return () => {
      active = false
      eventSource.close()
    }
  }, [initialSnapshot])

  return { snapshot, history, status, error }
}
