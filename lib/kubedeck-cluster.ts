export const KUBEDECK_SCHEMA_VERSION = "kubedeck.io/v1alpha1"

export type ClusterCategory =
  | "web-applications"
  | "databases-storage"
  | "observability-metrics"
  | "automation-workflows"
  | "deployments"
  | "ai-services"
  | "messaging-events"
  | "developer-tools"
  | "platform-security"
  | "other"

export type ClusterPort = {
  name?: string
  protocol: string
  port: number
  targetPort?: string
  nodePort?: number
}

export type ClusterNode = {
  name: string
  role: string
  ready: boolean
  status: string
  internalIP?: string
  hostname?: string
  kubeletVersion: string
  osImage: string
  operatingSystem: string
  architecture: string
  containerRuntime: string
  createdAt: string
  uptimeSeconds: number
  lastHeartbeatAt?: string
  capacity: {
    cpuMilli: number
    memoryBytes: number
    ephemeralStorageBytes: number
    pods: number
  }
  usage: {
    cpuMilli: number
    cpuPercent: number
    memoryBytes: number
    memoryPercent: number
    pods: number
    podAllocationPercent: number
    ephemeralStorageRequestedBytes: number
    ephemeralStorageRequestPercent: number
    metricsAvailable: boolean
  }
  conditions: Array<{
    type: string
    status: string
    reason?: string
    message?: string
    lastTransitionTime?: string
  }>
  unschedulable: boolean
  podCIDR?: string
  labels?: Record<string, string>
}

export type ClusterPod = {
  uid: string
  namespace: string
  name: string
  nodeName?: string
  phase: string
  status: string
  reason?: string
  ready: boolean
  restarts: number
  createdAt: string
  startedAt?: string
  ownerKind?: string
  ownerName?: string
  podIP?: string
  hostIP?: string
  containers: Array<{
    name: string
    image: string
    ready: boolean
    restartCount: number
    state: string
    startedAt?: string
  }>
  usage: {
    cpuMilli: number
    memoryBytes: number
    metricsAvailable: boolean
  }
  labels?: Record<string, string>
}

export type ClusterWorkload = {
  uid: string
  kind: string
  namespace: string
  name: string
  desired: number
  ready: number
  available: number
  status: string
  createdAt: string
  lastDeployedAt?: string
  selector?: Record<string, string>
  labels?: Record<string, string>
}

export type ClusterObjectReference = {
  kind: string
  namespace: string
  name: string
}

export type ClusterService = {
  uid: string
  namespace: string
  name: string
  category: ClusterCategory | string
  type: string
  status: string
  clusterDNS: string
  clusterIP?: string
  externalIPs: string[]
  externalURLs: string[]
  ports: ClusterPort[]
  selector?: Record<string, string>
  readyEndpoints: number
  totalEndpoints: number
  readyPods: number
  totalPods: number
  workloads?: ClusterObjectReference[]
  lastDeployedAt?: string
  uptimeSeconds: number
  labels?: Record<string, string>
  annotations?: Record<string, string>
}

export type ClusterIngress = {
  uid: string
  namespace: string
  name: string
  className?: string
  routes: Array<{
    host?: string
    path: string
    pathType?: string
    serviceName: string
    servicePort: string
    url?: string
  }>
  tlsHosts?: string[]
  addresses?: string[]
  createdAt: string
}

export type ClusterVolume = {
  uid: string
  namespace?: string
  name: string
  kind: string
  status: string
  storageClass?: string
  capacityBytes: number
  accessModes?: string[]
  volumeName?: string
  createdAt: string
  labels?: Record<string, string>
}

export type ClusterEvent = {
  uid: string
  namespace?: string
  name: string
  type: string
  reason?: string
  message?: string
  regardingKind?: string
  regardingNamespace?: string
  regardingName?: string
  count: number
  firstSeenAt?: string
  lastSeenAt?: string
}

export type ClusterSnapshot = {
  schemaVersion: typeof KUBEDECK_SCHEMA_VERSION
  generatedAt: string
  cluster: {
    id: string
    name: string
    kubernetesVersion: string
    platform?: string
  }
  summary: {
    nodes: number
    readyNodes: number
    namespaces: number
    pods: number
    readyPods: number
    workloads: number
    readyWorkloads: number
    services: number
    readyServices: number
    ingresses: number
    persistentVolumes: number
    warningEvents: number
  }
  dns: {
    provider: string
    serviceName?: string
    serviceDNS?: string
    serviceIP?: string
    clusterDomain: string
    ports: ClusterPort[]
    searchPath: string[]
    ready: boolean
    readyEndpoints?: number
    totalEndpoints?: number
  }
  nodes: ClusterNode[]
  pods: ClusterPod[]
  workloads: ClusterWorkload[]
  services: ClusterService[]
  ingresses: ClusterIngress[]
  volumes: ClusterVolume[]
  events: ClusterEvent[]
}

export type ClusterStreamEnvelope<T = unknown> = {
  id: number
  name: string
  clusterId: string
  sentAt: string
  data: T
}

export type LiveNodeView = {
  id: string
  name: string
  cluster: string
  role: "control-plane" | "worker"
  status: "ready" | "attention"
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
  metricsAvailable: boolean
}

export type ResourceTrendPoint = {
  time: string
  cpu: number
  memory: number
  storage: number
  pods: number
}

export type SnapshotHistory = ClusterSnapshot[]

const clusterCategories = new Set<ClusterCategory>([
  "web-applications",
  "databases-storage",
  "observability-metrics",
  "automation-workflows",
  "deployments",
  "ai-services",
  "messaging-events",
  "developer-tools",
  "platform-security",
  "other",
])

export function parseClusterSnapshot(value: unknown): ClusterSnapshot {
  if (!isRecord(value)) throw new Error("Cluster snapshot must be an object.")
  if (value.schemaVersion !== KUBEDECK_SCHEMA_VERSION) {
    throw new Error(
      `Unsupported cluster snapshot schema: ${String(value.schemaVersion)}`
    )
  }
  if (!isRecord(value.cluster) || typeof value.cluster.id !== "string") {
    throw new Error("Cluster snapshot is missing cluster identity.")
  }
  if (!isRecord(value.summary) || !isRecord(value.dns)) {
    throw new Error("Cluster snapshot is missing summary or DNS data.")
  }
  for (const field of [
    "nodes",
    "pods",
    "workloads",
    "services",
    "ingresses",
    "volumes",
    "events",
  ] as const) {
    if (!Array.isArray(value[field])) {
      throw new Error(`Cluster snapshot field ${field} must be an array.`)
    }
  }
  return value as ClusterSnapshot
}

export function parseClusterStreamEnvelope(
  value: unknown
): ClusterStreamEnvelope {
  if (
    !isRecord(value) ||
    typeof value.id !== "number" ||
    typeof value.name !== "string" ||
    typeof value.clusterId !== "string" ||
    !("data" in value)
  ) {
    throw new Error("Invalid KubeDeck event envelope.")
  }
  return value as ClusterStreamEnvelope
}

export function appendSnapshotHistory(
  history: SnapshotHistory,
  snapshot: ClusterSnapshot,
  limit = 24
): SnapshotHistory {
  const withoutDuplicate = history.filter(
    (item) => item.generatedAt !== snapshot.generatedAt
  )
  return [...withoutDuplicate, snapshot].slice(-Math.max(1, limit))
}

export function buildLiveNodes(
  snapshot: ClusterSnapshot,
  now?: number
): LiveNodeView[] {
  const referenceTime =
    now ??
    (Number.isFinite(Date.parse(snapshot.generatedAt))
      ? Date.parse(snapshot.generatedAt)
      : Date.now())
  return snapshot.nodes.map((node) => {
    const pressure = node.conditions.find(
      (condition) =>
        condition.type !== "Ready" &&
        condition.status.toLowerCase() === "true"
    )
    const operationallyReady =
      node.ready &&
      !node.unschedulable &&
      !pressure &&
      node.status.toLowerCase() === "ready"
    return {
      id: node.name,
      name: node.name,
      cluster: snapshot.cluster.name,
      role: node.role === "control-plane" ? "control-plane" : "worker",
      status: operationallyReady ? "ready" : "attention",
      cpu: roundedPercent(node.usage.cpuPercent),
      memory: roundedPercent(node.usage.memoryPercent),
      storage: roundedPercent(node.usage.ephemeralStorageRequestPercent),
      podAllocation: roundedPercent(node.usage.podAllocationPercent),
      pods: `${node.usage.pods} / ${node.capacity.pods}`,
      internalIp: node.internalIP || "Unavailable",
      kubeletVersion: node.kubeletVersion || "Unknown",
      age: formatDuration(node.uptimeSeconds),
      heartbeat: relativeTimestamp(node.lastHeartbeatAt, referenceTime),
      condition: node.unschedulable
        ? "Unschedulable"
        : humanize(pressure?.reason || pressure?.type || node.status),
      metricsAvailable: node.usage.metricsAvailable,
    }
  })
}

export function buildResourceTrend(
  history: SnapshotHistory,
  nodeId: string
): ResourceTrendPoint[] {
  return history.map((snapshot) => {
    const nodes =
      nodeId === "fleet"
        ? snapshot.nodes
        : snapshot.nodes.filter((node) => node.name === nodeId)
    return {
      time: formatTrendTime(snapshot.generatedAt),
      cpu: average(nodes.map((node) => node.usage.cpuPercent)),
      memory: average(nodes.map((node) => node.usage.memoryPercent)),
      storage: average(
        nodes.map((node) => node.usage.ephemeralStorageRequestPercent)
      ),
      pods: average(nodes.map((node) => node.usage.podAllocationPercent)),
    }
  })
}

export function normalizeClusterCategory(value: string): ClusterCategory {
  return clusterCategories.has(value as ClusterCategory)
    ? (value as ClusterCategory)
    : "other"
}

export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "Not running"
  const days = Math.floor(seconds / 86_400)
  const hours = Math.floor((seconds % 86_400) / 3_600)
  const minutes = Math.floor((seconds % 3_600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${Math.max(1, minutes)}m`
}

export function relativeTimestamp(
  value: string | undefined,
  now = Date.now()
): string {
  if (!value) return "Unavailable"
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return "Unavailable"
  const seconds = Math.max(0, Math.floor((now - timestamp) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  if (seconds < 3_600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86_400) return `${Math.floor(seconds / 3_600)}h ago`
  return `${Math.floor(seconds / 86_400)}d ago`
}

export function humanize(value: string): string {
  const spaced = value
    .replace(/[-_]+/gu, " ")
    .replace(/([a-z0-9])([A-Z])/gu, "$1 $2")
    .trim()
  if (!spaced) return "Unknown"
  return spaced.charAt(0).toUpperCase() + spaced.slice(1)
}

function formatTrendTime(timestamp: string) {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return "--:--"
  return `${String(date.getUTCHours()).padStart(2, "0")}:${String(
    date.getUTCMinutes()
  ).padStart(2, "0")}:${String(date.getUTCSeconds()).padStart(2, "0")} UTC`
}

function average(values: number[]) {
  const finite = values.filter(Number.isFinite)
  if (finite.length === 0) return 0
  return roundedPercent(
    finite.reduce((total, value) => total + value, 0) / finite.length
  )
}

function roundedPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.round(Math.max(0, Math.min(100, value)) * 10) / 10
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value)
}
