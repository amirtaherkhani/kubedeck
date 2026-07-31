export type NodeRole = "control-plane" | "worker"
export type NodeStatus = "ready" | "attention"

export type NodeSnapshot = {
  id: string
  name: string
  cluster: string
  role: NodeRole
  status: NodeStatus
  cpu: number
  memory: number
  storage: number
  podAllocation: number
  pods: string
  internalIp: string
  kubeletVersion: string
  age: string
  heartbeat: string
  condition: string
}

export type ResourceTrendPoint = {
  time: string
  cpu: number
  memory: number
  storage: number
  pods: number
}

export const monitoringSnapshot = {
  capturedAt: "Jul 29, 2026 · 05:21 UTC",
  mode: "Illustrative fallback snapshot",
  source: "bundled fallback inventory",
  liveIntegration: "kubedeck-agent",
} as const

export const nodeSnapshots: NodeSnapshot[] = [
  {
    id: "control-plane-01",
    name: "control-plane-01",
    cluster: "sample-primary",
    role: "control-plane",
    status: "ready",
    cpu: 28,
    memory: 44,
    storage: 35,
    podAllocation: 57,
    pods: "17 / 30",
    internalIp: "10.0.0.11",
    kubeletVersion: "v1.31.2",
    age: "28d",
    heartbeat: "32s ago",
    condition: "Ready",
  },
  {
    id: "worker-apps-01",
    name: "worker-apps-01",
    cluster: "sample-primary",
    role: "worker",
    status: "ready",
    cpu: 52,
    memory: 68,
    storage: 49,
    podAllocation: 75,
    pods: "27 / 36",
    internalIp: "10.0.0.21",
    kubeletVersion: "v1.31.2",
    age: "21d",
    heartbeat: "28s ago",
    condition: "Ready",
  },
  {
    id: "worker-data-01",
    name: "worker-data-01",
    cluster: "sample-primary",
    role: "worker",
    status: "attention",
    cpu: 64,
    memory: 82,
    storage: 73,
    podAllocation: 75,
    pods: "18 / 24",
    internalIp: "10.0.0.31",
    kubeletVersion: "v1.31.2",
    age: "21d",
    heartbeat: "3m ago",
    condition: "Memory pressure",
  },
]

export const fleetResourceTrend: ResourceTrendPoint[] = [
  { time: "04:00", cpu: 31, memory: 55, storage: 49, pods: 62 },
  { time: "04:10", cpu: 36, memory: 57, storage: 49, pods: 63 },
  { time: "04:20", cpu: 33, memory: 58, storage: 50, pods: 64 },
  { time: "04:30", cpu: 41, memory: 60, storage: 50, pods: 65 },
  { time: "04:40", cpu: 54, memory: 62, storage: 51, pods: 67 },
  { time: "04:50", cpu: 57, memory: 64, storage: 51, pods: 68 },
  { time: "05:00", cpu: 44, memory: 65, storage: 52, pods: 69 },
  { time: "05:10", cpu: 48, memory: 65, storage: 52, pods: 69 },
]

const trendOffsets = {
  cpu: [-11, -6, -9, -4, 6, 9, -3, 0],
  memory: [-8, -7, -6, -5, -3, -2, -1, 0],
  storage: [-3, -3, -2, -2, -1, -1, 0, 0],
  pods: [-7, -6, -5, -4, -2, -1, 0, 0],
} as const

export function getNodeResourceTrend(nodeId: string): ResourceTrendPoint[] {
  if (nodeId === "fleet") return fleetResourceTrend

  const node = nodeSnapshots.find((item) => item.id === nodeId)
  if (!node) return fleetResourceTrend

  return fleetResourceTrend.map((point, index) => ({
    time: point.time,
    cpu: clampPercentage(node.cpu + trendOffsets.cpu[index]),
    memory: clampPercentage(node.memory + trendOffsets.memory[index]),
    storage: clampPercentage(node.storage + trendOffsets.storage[index]),
    pods: clampPercentage(node.podAllocation + trendOffsets.pods[index]),
  }))
}

export const nodeSummary = {
  total: nodeSnapshots.length,
  ready: nodeSnapshots.filter((node) => node.status === "ready").length,
  attention: nodeSnapshots.filter((node) => node.status === "attention").length,
  pods: nodeSnapshots.reduce(
    (total, node) => total + Number(node.pods.split(" / ")[0]),
    0
  ),
  podCapacity: nodeSnapshots.reduce(
    (total, node) => total + Number(node.pods.split(" / ")[1]),
    0
  ),
} as const

function clampPercentage(value: number) {
  return Math.max(0, Math.min(100, value))
}
