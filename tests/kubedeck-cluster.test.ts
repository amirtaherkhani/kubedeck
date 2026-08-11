import assert from "node:assert/strict"
import test from "node:test"

import {
  appendSnapshotHistory,
  buildLiveNodes,
  buildNodeIngressSummaries,
  buildResourceTrend,
  formatDuration,
  humanize,
  normalizeClusterCategory,
  parseClusterSnapshot,
  parseClusterStreamEnvelope,
  relativeTimestamp,
  type ClusterSnapshot,
} from "../lib/kubedeck-cluster.ts"

function snapshot(
  generatedAt = "2026-07-31T12:00:00Z",
  cpu = 25
): ClusterSnapshot {
  return {
    schemaVersion: "kubedeck.io/v1alpha1",
    generatedAt,
    cluster: {
      id: "test",
      name: "Test cluster",
      kubernetesVersion: "v1.36.3",
      platform: "linux/arm64",
    },
    summary: {
      nodes: 2,
      readyNodes: 2,
      namespaces: 3,
      pods: 4,
      readyPods: 3,
      workloads: 2,
      readyWorkloads: 1,
      services: 2,
      readyServices: 1,
      ingresses: 1,
      persistentVolumes: 2,
      warningEvents: 1,
    },
    dns: {
      provider: "CoreDNS",
      serviceName: "kube-system/kube-dns",
      serviceDNS: "kube-dns.kube-system.svc.cluster.local",
      serviceIP: "10.43.0.10",
      clusterDomain: "cluster.local",
      ports: [{ protocol: "UDP", port: 53 }],
      searchPath: [
        "<namespace>.svc.cluster.local",
        "svc.cluster.local",
        "cluster.local",
      ],
      ready: true,
      readyEndpoints: 1,
      totalEndpoints: 1,
    },
    nodes: [
      {
        name: "control-plane-01",
        role: "control-plane",
        ready: true,
        status: "Ready",
        internalIP: "10.0.0.1",
        kubeletVersion: "v1.36.3",
        osImage: "Linux",
        operatingSystem: "linux",
        architecture: "arm64",
        containerRuntime: "containerd://2",
        createdAt: "2026-07-30T12:00:00Z",
        uptimeSeconds: 86_400,
        lastHeartbeatAt: "2026-07-31T11:59:30Z",
        capacity: {
          cpuMilli: 4_000,
          memoryBytes: 8_000,
          ephemeralStorageBytes: 10_000,
          pods: 20,
        },
        usage: {
          cpuMilli: 1_000,
          cpuPercent: cpu,
          memoryBytes: 4_000,
          memoryPercent: 50,
          pods: 5,
          podAllocationPercent: 25,
          ephemeralStorageRequestedBytes: 1_000,
          ephemeralStorageRequestPercent: 10,
          metricsAvailable: true,
        },
        conditions: [{ type: "Ready", status: "True" }],
        unschedulable: false,
      },
      {
        name: "worker-pressure",
        role: "worker",
        ready: true,
        status: "KubeletHasInsufficientMemory",
        internalIP: "10.0.0.2",
        kubeletVersion: "v1.36.3",
        osImage: "Linux",
        operatingSystem: "linux",
        architecture: "arm64",
        containerRuntime: "containerd://2",
        createdAt: "2026-07-30T12:00:00Z",
        uptimeSeconds: 86_400,
        lastHeartbeatAt: "2026-07-31T11:58:00Z",
        capacity: {
          cpuMilli: 2_000,
          memoryBytes: 4_000,
          ephemeralStorageBytes: 10_000,
          pods: 10,
        },
        usage: {
          cpuMilli: 1_000,
          cpuPercent: 50,
          memoryBytes: 3_600,
          memoryPercent: 90,
          pods: 8,
          podAllocationPercent: 80,
          ephemeralStorageRequestedBytes: 4_000,
          ephemeralStorageRequestPercent: 40,
          metricsAvailable: true,
        },
        conditions: [
          { type: "Ready", status: "True" },
          {
            type: "MemoryPressure",
            status: "True",
            reason: "KubeletHasInsufficientMemory",
          },
        ],
        unschedulable: false,
      },
    ],
    pods: [],
    workloads: [],
    services: [],
    ingresses: [],
    volumes: [],
    events: [],
  }
}

test("validates the complete top-level agent snapshot contract", () => {
  const valid = snapshot()
  assert.equal(parseClusterSnapshot(valid), valid)
  assert.throws(
    () => parseClusterSnapshot({ ...valid, schemaVersion: "legacy" }),
    /Unsupported cluster snapshot schema/
  )
  assert.throws(
    () => parseClusterSnapshot({ ...valid, services: null }),
    /services must be an array/
  )
  assert.throws(
    () => parseClusterSnapshot({ schemaVersion: "kubedeck.io/v1alpha1" }),
    /cluster identity/
  )
})

test("maps ingress routes to the nodes running their selected backend pods", () => {
  const live = snapshot()
  live.pods = [
    {
      uid: "pod-control-plane",
      namespace: "apps",
      name: "api-control-plane",
      nodeName: "control-plane-01",
      phase: "Running",
      status: "ready",
      ready: true,
      restarts: 0,
      createdAt: live.generatedAt,
      containers: [],
      usage: { cpuMilli: 0, memoryBytes: 0, metricsAvailable: false },
      labels: { app: "api" },
    },
    {
      uid: "pod-worker",
      namespace: "apps",
      name: "api-worker",
      nodeName: "worker-pressure",
      phase: "Running",
      status: "attention",
      ready: false,
      restarts: 1,
      createdAt: live.generatedAt,
      containers: [],
      usage: { cpuMilli: 0, memoryBytes: 0, metricsAvailable: false },
      labels: { app: "api" },
    },
  ]
  live.services = [
    {
      uid: "service-api",
      namespace: "apps",
      name: "api",
      category: "web-applications",
      type: "ClusterIP",
      status: "ready",
      clusterDNS: "api.apps.svc.cluster.local",
      externalIPs: [],
      externalURLs: ["https://api.example.test"],
      ports: [{ protocol: "TCP", port: 80 }],
      selector: { app: "api" },
      readyEndpoints: 1,
      totalEndpoints: 2,
      readyPods: 1,
      totalPods: 2,
      workloads: [],
      uptimeSeconds: 3600,
    },
  ]
  live.ingresses = [
    {
      uid: "ingress-api",
      namespace: "apps",
      name: "api",
      routes: [
        {
          host: "api.example.test",
          path: "/",
          serviceName: "api",
          servicePort: "80",
        },
        {
          host: "api.internal.test",
          path: "/internal",
          serviceName: "api",
          servicePort: "80",
        },
      ],
      createdAt: live.generatedAt,
    },
  ]

  assert.deepEqual(
    buildNodeIngressSummaries(live).map((node) => ({
      name: node.name,
      ingressCount: node.ingressCount,
      routeCount: node.routeCount,
      serviceCount: node.serviceCount,
      readyBackendPods: node.readyBackendPods,
      totalBackendPods: node.totalBackendPods,
      hosts: node.hosts,
    })),
    [
      {
        name: "control-plane-01",
        ingressCount: 1,
        routeCount: 2,
        serviceCount: 1,
        readyBackendPods: 1,
        totalBackendPods: 1,
        hosts: ["api.example.test", "api.internal.test"],
      },
      {
        name: "worker-pressure",
        ingressCount: 1,
        routeCount: 2,
        serviceCount: 1,
        readyBackendPods: 0,
        totalBackendPods: 1,
        hosts: ["api.example.test", "api.internal.test"],
      },
    ]
  )
})

test("maps readiness, pressure, metrics, age, and heartbeat into node views", () => {
  const nodes = buildLiveNodes(snapshot(), Date.parse("2026-07-31T12:00:00Z"))
  assert.deepEqual(
    nodes.map((node) => ({
      id: node.id,
      role: node.role,
      status: node.status,
      condition: node.condition,
      heartbeat: node.heartbeat,
      pods: node.pods,
    })),
    [
      {
        id: "control-plane-01",
        role: "control-plane",
        status: "ready",
        condition: "Ready",
        heartbeat: "30s ago",
        pods: "5 / 20",
      },
      {
        id: "worker-pressure",
        role: "worker",
        status: "attention",
        condition: "Kubelet Has Insufficient Memory",
        heartbeat: "2m ago",
        pods: "8 / 10",
      },
    ]
  )
})

test("keeps a bounded deduplicated SSE history and derives live chart points", () => {
  let history: ClusterSnapshot[] = []
  history = appendSnapshotHistory(history, snapshot("2026-07-31T12:00:00Z", 20), 2)
  history = appendSnapshotHistory(history, snapshot("2026-07-31T12:00:10Z", 30), 2)
  history = appendSnapshotHistory(history, snapshot("2026-07-31T12:00:10Z", 40), 2)
  history = appendSnapshotHistory(history, snapshot("2026-07-31T12:00:20Z", 60), 2)
  assert.deepEqual(
    history.map((item) => item.generatedAt),
    ["2026-07-31T12:00:10Z", "2026-07-31T12:00:20Z"]
  )
  assert.deepEqual(
    buildResourceTrend(history, "control-plane-01").map((point) => point.cpu),
    [40, 60]
  )
  assert.deepEqual(
    buildResourceTrend(history, "fleet").map((point) => point.cpu),
    [45, 55]
  )
})

test("parses stream envelopes and formats dashboard-safe values", () => {
  const envelope = parseClusterStreamEnvelope({
    id: 7,
    name: "snapshot",
    clusterId: "test",
    sentAt: "2026-07-31T12:00:00Z",
    data: snapshot(),
  })
  assert.equal(envelope.id, 7)
  assert.throws(() => parseClusterStreamEnvelope({ id: "7" }), /Invalid/)
  assert.equal(formatDuration(90_000), "1d 1h")
  assert.equal(formatDuration(0), "Not running")
  assert.equal(relativeTimestamp(undefined), "Unavailable")
  assert.equal(humanize("MemoryPressure"), "Memory Pressure")
  assert.equal(normalizeClusterCategory("unexpected"), "other")
})
