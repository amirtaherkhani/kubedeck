"use client"

import * as React from "react"
import type { LucideIcon } from "lucide-react"
import {
  ActivityIcon,
  ArrowDownAZIcon,
  ArrowUpRightIcon,
  BarChart3Icon,
  BoxesIcon,
  BracesIcon,
  CableIcon,
  CheckIcon,
  CircleCheckIcon,
  CircleDotDashedIcon,
  CircleGaugeIcon,
  Clock3Icon,
  CloudCogIcon,
  CopyIcon,
  DatabaseIcon,
  ExternalLinkIcon,
  EyeIcon,
  GitBranchIcon,
  Globe2Icon,
  HardDriveIcon,
  InboxIcon,
  KeyRoundIcon,
  Layers3Icon,
  MailIcon,
  MessageSquareIcon,
  NetworkIcon,
  RefreshCwIcon,
  SearchIcon,
  ServerCogIcon,
  Settings2Icon,
  SlidersHorizontalIcon,
  SparklesIcon,
  WorkflowIcon,
  XIcon,
  ZapIcon,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@/components/ui/toggle-group"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

type CatalogStatus = "ready" | "attention"
type CatalogKind = "app" | "service"
type SortMode = "recommended" | "name" | "namespace"

type CatalogItem = {
  id: string
  name: string
  kind: CatalogKind
  namespace: string
  summary: string
  endpoint: string
  href?: string
  ports: string[]
  protocol: string
  workload: string
  pods: string
  source: string
  status: CatalogStatus
  icon: LucideIcon
}

const webApps: CatalogItem[] = [
  {
    id: "grafana",
    name: "Grafana",
    kind: "app",
    namespace: "monitoring",
    summary: "Dashboards for metrics, logs, traces, and alerts.",
    endpoint: "grafana.dev.local",
    href: "https://grafana.dev.local",
    ports: ["80 → 3000"],
    protocol: "HTTPS",
    workload: "Deployment/grafana",
    pods: "1 / 1 ready",
    source: "Ingress/grafana",
    status: "ready",
    icon: BarChart3Icon,
  },
  {
    id: "radar",
    name: "Radar",
    kind: "app",
    namespace: "radar",
    summary: "Live Kubernetes topology and service traffic.",
    endpoint: "radar.dev.local",
    href: "https://radar.dev.local",
    ports: ["9280"],
    protocol: "HTTPS",
    workload: "Deployment/radar",
    pods: "1 / 1 ready",
    source: "Ingress/radar",
    status: "ready",
    icon: NetworkIcon,
  },
  {
    id: "infisical",
    name: "Infisical",
    kind: "app",
    namespace: "infisical",
    summary: "Secrets, environments, and access management.",
    endpoint: "infisical.dev.local",
    href: "https://infisical.dev.local",
    ports: ["8080"],
    protocol: "HTTPS",
    workload: "Deployment/infisical-backend",
    pods: "1 / 1 ready",
    source: "Ingress/infisical-ingress",
    status: "ready",
    icon: KeyRoundIcon,
  },
  {
    id: "n8n",
    name: "n8n",
    kind: "app",
    namespace: "n8n",
    summary: "Workflow automation and integration builder.",
    endpoint: "n8n.dev.local",
    href: "https://n8n.dev.local",
    ports: ["80 → 5678"],
    protocol: "HTTPS",
    workload: "Deployment/n8n",
    pods: "1 / 1 ready",
    source: "Ingress/n8n",
    status: "ready",
    icon: WorkflowIcon,
  },
  {
    id: "temporal",
    name: "Temporal",
    kind: "app",
    namespace: "temporal",
    summary: "Workflow execution, history, and task visibility.",
    endpoint: "temporal.dev.local",
    href: "https://temporal.dev.local",
    ports: ["8080"],
    protocol: "HTTPS",
    workload: "Deployment/temporal-web",
    pods: "1 / 1 ready",
    source: "Ingress/temporal-web",
    status: "ready",
    icon: Clock3Icon,
  },
  {
    id: "nats-ui",
    name: "NATS UI",
    kind: "app",
    namespace: "storage",
    summary: "Streams, consumers, messages, and server state.",
    endpoint: "nats-ui.dev.local",
    href: "https://nats-ui.dev.local",
    ports: ["80 → 31311"],
    protocol: "HTTPS",
    workload: "Deployment/nats-ui",
    pods: "1 / 1 ready",
    source: "Ingress/nats-ui",
    status: "ready",
    icon: ZapIcon,
  },
  {
    id: "kafka-ui",
    name: "Kafka UI",
    kind: "app",
    namespace: "storage",
    summary: "Topics, brokers, schemas, and consumer groups.",
    endpoint: "kafka-ui.dev.local",
    href: "https://kafka-ui.dev.local",
    ports: ["80 → 8080"],
    protocol: "HTTPS",
    workload: "Deployment/kafka-ui",
    pods: "1 / 1 ready",
    source: "Ingress/kafka-ui",
    status: "ready",
    icon: GitBranchIcon,
  },
  {
    id: "minio-console",
    name: "MinIO Console",
    kind: "app",
    namespace: "storage",
    summary: "Buckets, objects, policies, and storage health.",
    endpoint: "s3-ui.dev.local",
    href: "https://s3-ui.dev.local",
    ports: ["9001"],
    protocol: "HTTPS",
    workload: "StatefulSet/minio",
    pods: "1 / 1 ready",
    source: "Ingress/minio-console",
    status: "ready",
    icon: HardDriveIcon,
  },
  {
    id: "pgadmin",
    name: "pgAdmin",
    kind: "app",
    namespace: "storage",
    summary: "PostgreSQL administration and query workspace.",
    endpoint: "pgadmin.dev.local",
    href: "https://pgadmin.dev.local",
    ports: ["80 → 5050"],
    protocol: "HTTPS",
    workload: "Deployment/pgadmin",
    pods: "1 / 1 ready",
    source: "Ingress/pgadmin",
    status: "ready",
    icon: DatabaseIcon,
  },
  {
    id: "redis-commander",
    name: "Redis Commander",
    kind: "app",
    namespace: "storage",
    summary: "Browse keys, values, and Redis database state.",
    endpoint: "redis-ui.dev.local",
    href: "https://redis-ui.dev.local",
    ports: ["80 → 8081"],
    protocol: "HTTPS",
    workload: "Deployment/redis-commander",
    pods: "1 / 1 ready",
    source: "Ingress/redis-commander",
    status: "ready",
    icon: Layers3Icon,
  },
  {
    id: "rabbitmq",
    name: "RabbitMQ",
    kind: "app",
    namespace: "storage",
    summary: "Queues, exchanges, bindings, and broker health.",
    endpoint: "rabbitmq-ui.dev.local",
    href: "https://rabbitmq-ui.dev.local",
    ports: ["15672"],
    protocol: "HTTPS",
    workload: "StatefulSet/rabbitmq",
    pods: "1 / 1 ready",
    source: "Ingress/rabbitmq",
    status: "ready",
    icon: MessageSquareIcon,
  },
  {
    id: "mailpit",
    name: "Mailpit",
    kind: "app",
    namespace: "storage",
    summary: "Local email inbox and SMTP test console.",
    endpoint: "mailpit-ui.dev.local",
    href: "https://mailpit-ui.dev.local",
    ports: ["80 → 8025"],
    protocol: "HTTPS",
    workload: "Deployment/mailpit",
    pods: "1 / 1 ready",
    source: "Ingress/mailpit-ui",
    status: "ready",
    icon: MailIcon,
  },
  {
    id: "mongo-ui",
    name: "Mongoku",
    kind: "app",
    namespace: "storage",
    summary: "MongoDB collections, documents, and queries.",
    endpoint: "mongo-ui.dev.local",
    href: "https://mongo-ui.dev.local",
    ports: ["80 → 3100"],
    protocol: "HTTPS",
    workload: "Deployment/mongoku",
    pods: "1 / 1 ready",
    source: "Ingress/mongoku",
    status: "ready",
    icon: BracesIcon,
  },
  {
    id: "schema-registry-ui",
    name: "Schema Registry UI",
    kind: "app",
    namespace: "storage",
    summary: "Inspect and manage event schema subjects.",
    endpoint: "schema-registry-ui.dev.local",
    href: "https://schema-registry-ui.dev.local",
    ports: ["80 → 8080"],
    protocol: "HTTPS",
    workload: "Deployment/schema-registry-ui",
    pods: "1 / 1 ready",
    source: "Ingress/schema-registry-ui",
    status: "ready",
    icon: BoxesIcon,
  },
  {
    id: "jaeger-ui",
    name: "Jaeger UI",
    kind: "app",
    namespace: "storage",
    summary: "Trace search, spans, and dependency inspection.",
    endpoint: "jaeger-ui.dev.local",
    href: "https://jaeger-ui.dev.local",
    ports: ["80 → 16686"],
    protocol: "HTTPS",
    workload: "Deployment/jaeger",
    pods: "1 / 1 ready",
    source: "Ingress/jaeger-ui",
    status: "ready",
    icon: EyeIcon,
  },
  {
    id: "grpcui",
    name: "gRPC UI",
    kind: "app",
    namespace: "storage",
    summary: "Explore and invoke local gRPC endpoints.",
    endpoint: "grpcui.dev.local",
    href: "https://grpcui.dev.local",
    ports: ["80 → 8080"],
    protocol: "HTTPS",
    workload: "Deployment/grpcui",
    pods: "1 / 1 ready",
    source: "Ingress/grpcui",
    status: "ready",
    icon: CableIcon,
  },
  {
    id: "k6-dashboard",
    name: "k6 Dashboard",
    kind: "app",
    namespace: "k6-tests",
    summary: "Ephemeral live dashboard for the active load test.",
    endpoint: "k6-dashboard.dev.local",
    href: "https://k6-dashboard.dev.local",
    ports: ["5665"],
    protocol: "HTTPS",
    workload: "Pod/k6-web-dashboard",
    pods: "0 endpoints",
    source: "Ingress/k6-web-dashboard",
    status: "attention",
    icon: CircleGaugeIcon,
  },
]

const services: CatalogItem[] = [
  {
    id: "finance-api",
    name: "VeroVault Finance",
    kind: "service",
    namespace: "vero-vault-finance",
    summary: "Finance API and operational endpoints.",
    endpoint: "vero-vault-finance.dev.local",
    href: "https://vero-vault-finance.dev.local",
    ports: ["3000"],
    protocol: "HTTPS",
    workload: "Deployment/vero-vault-finance",
    pods: "2 / 2 ready",
    source: "Ingress/vero-vault-finance",
    status: "ready",
    icon: SparklesIcon,
  },
  {
    id: "dev-io-mcp",
    name: "dev.io MCP",
    kind: "service",
    namespace: "dev-io",
    summary: "Developer publishing and MCP service endpoint.",
    endpoint: "dev-io-mcp.dev.local",
    href: "https://dev-io-mcp.dev.local",
    ports: ["3000"],
    protocol: "HTTPS",
    workload: "Deployment/dev-io-mcp",
    pods: "1 / 1 ready",
    source: "Ingress/dev-io-mcp",
    status: "ready",
    icon: CloudCogIcon,
  },
  {
    id: "diago",
    name: "Diago",
    kind: "service",
    namespace: "diago",
    summary: "Engineering diagram MCP and rendering service.",
    endpoint: "diago.dev.local",
    href: "https://diago.dev.local",
    ports: ["3000"],
    protocol: "HTTPS",
    workload: "Deployment/diago",
    pods: "1 / 1 ready",
    source: "Ingress/diago",
    status: "ready",
    icon: BoxesIcon,
  },
  {
    id: "postgresql",
    name: "PostgreSQL",
    kind: "service",
    namespace: "storage",
    summary: "Shared relational database for platform workloads.",
    endpoint: "postgresql.storage.svc.cluster.local",
    ports: ["5432"],
    protocol: "TCP",
    workload: "StatefulSet/postgresql",
    pods: "1 / 1 ready",
    source: "Service/postgresql",
    status: "ready",
    icon: DatabaseIcon,
  },
  {
    id: "redis",
    name: "Redis",
    kind: "service",
    namespace: "storage",
    summary: "Shared cache, sessions, and short-lived state.",
    endpoint: "redis.storage.svc.cluster.local",
    ports: ["6379"],
    protocol: "TCP",
    workload: "StatefulSet/redis",
    pods: "1 / 1 ready",
    source: "Service/redis",
    status: "ready",
    icon: Layers3Icon,
  },
  {
    id: "mongodb",
    name: "MongoDB",
    kind: "service",
    namespace: "storage",
    summary: "Document database for local application data.",
    endpoint: "mongodb.storage.svc.cluster.local",
    ports: ["27017"],
    protocol: "TCP",
    workload: "StatefulSet/mongodb",
    pods: "1 / 1 ready",
    source: "Service/mongodb",
    status: "ready",
    icon: BracesIcon,
  },
  {
    id: "nats",
    name: "NATS JetStream",
    kind: "service",
    namespace: "storage",
    summary: "Durable messaging, streams, and consumers.",
    endpoint: "nats.storage.svc.cluster.local",
    ports: ["4222"],
    protocol: "NATS",
    workload: "StatefulSet/nats",
    pods: "3 / 3 ready",
    source: "Service/nats",
    status: "ready",
    icon: ZapIcon,
  },
  {
    id: "kafka",
    name: "Kafka",
    kind: "service",
    namespace: "storage",
    summary: "Event streaming broker for local integrations.",
    endpoint: "kafka.storage.svc.cluster.local",
    ports: ["9092", "9094"],
    protocol: "TCP",
    workload: "StatefulSet/kafka",
    pods: "1 / 1 ready",
    source: "Service/kafka",
    status: "ready",
    icon: GitBranchIcon,
  },
  {
    id: "prometheus",
    name: "Prometheus",
    kind: "service",
    namespace: "monitoring",
    summary: "Metrics store, query API, and remote-write receiver.",
    endpoint: "monitoring-kube-prometheus-prometheus.monitoring.svc.cluster.local",
    ports: ["9090"],
    protocol: "HTTP",
    workload: "StatefulSet/prometheus",
    pods: "1 / 1 ready",
    source: "Service/monitoring-kube-prometheus-prometheus",
    status: "ready",
    icon: ActivityIcon,
  },
  {
    id: "loki",
    name: "Loki",
    kind: "service",
    namespace: "monitoring",
    summary: "Cluster log storage and query endpoint.",
    endpoint: "loki.monitoring.svc.cluster.local",
    ports: ["3100", "9095"],
    protocol: "HTTP / gRPC",
    workload: "StatefulSet/loki",
    pods: "1 / 1 ready",
    source: "Service/loki",
    status: "ready",
    icon: InboxIcon,
  },
  {
    id: "tempo",
    name: "Tempo",
    kind: "service",
    namespace: "monitoring",
    summary: "Distributed trace storage and query endpoint.",
    endpoint: "tempo.monitoring.svc.cluster.local",
    ports: ["3200", "4317", "4318"],
    protocol: "HTTP / gRPC",
    workload: "StatefulSet/tempo",
    pods: "1 / 1 ready",
    source: "Service/tempo",
    status: "ready",
    icon: NetworkIcon,
  },
  {
    id: "alloy",
    name: "Grafana Alloy",
    kind: "service",
    namespace: "monitoring",
    summary: "OTLP receiver and telemetry collection pipeline.",
    endpoint: "alloy.monitoring.svc.cluster.local",
    ports: ["4317", "4318"],
    protocol: "OTLP",
    workload: "DaemonSet/alloy",
    pods: "1 endpoint",
    source: "Service/alloy",
    status: "ready",
    icon: ServerCogIcon,
  },
  {
    id: "temporal-frontend",
    name: "Temporal Frontend",
    kind: "service",
    namespace: "temporal",
    summary: "Workflow client and administration gateway.",
    endpoint: "temporal-frontend.temporal.svc.cluster.local",
    ports: ["7233", "7243"],
    protocol: "gRPC / HTTP",
    workload: "Deployment/temporal-frontend",
    pods: "1 / 1 ready",
    source: "Service/temporal-frontend",
    status: "ready",
    icon: WorkflowIcon,
  },
  {
    id: "minio-api",
    name: "MinIO API",
    kind: "service",
    namespace: "storage",
    summary: "S3-compatible object storage endpoint.",
    endpoint: "minio.storage.svc.cluster.local",
    ports: ["9000"],
    protocol: "S3 / HTTP",
    workload: "StatefulSet/minio",
    pods: "1 / 1 ready",
    source: "Service/minio",
    status: "ready",
    icon: HardDriveIcon,
  },
  {
    id: "schema-registry",
    name: "Schema Registry",
    kind: "service",
    namespace: "storage",
    summary: "Schema API for event producers and consumers.",
    endpoint: "schema-registry.storage.svc.cluster.local",
    ports: ["8080"],
    protocol: "HTTP",
    workload: "Deployment/schema-registry",
    pods: "1 / 1 ready",
    source: "Service/schema-registry",
    status: "ready",
    icon: BoxesIcon,
  },
]

const clusterStats = [
  { label: "Node", value: "1 ready", detail: "lima-rancher-desktop" },
  { label: "Namespaces", value: "18", detail: "active cluster scopes" },
  { label: "Workloads", value: "46", detail: "deployments + stateful sets" },
  { label: "Pods", value: "62 / 68", detail: "running and ready" },
] as const

function matchesSearch(item: CatalogItem, search: string) {
  const needle = search.trim().toLowerCase()
  if (!needle) return true

  return [
    item.name,
    item.namespace,
    item.summary,
    item.endpoint,
    item.protocol,
    item.ports.join(" "),
  ]
    .join(" ")
    .toLowerCase()
    .includes(needle)
}

function sortItems(items: CatalogItem[], sortMode: SortMode) {
  if (sortMode === "recommended") return items

  return [...items].sort((left, right) => {
    const leftValue = sortMode === "name" ? left.name : left.namespace
    const rightValue = sortMode === "name" ? right.name : right.namespace
    return leftValue.localeCompare(rightValue)
  })
}

function StatusBadge({ status }: { status: CatalogStatus }) {
  if (status === "attention") {
    return (
      <Badge variant="destructive">
        <CircleDotDashedIcon data-icon="inline-start" />
        Offline
      </Badge>
    )
  }

  return (
    <Badge variant="default">
      <CircleCheckIcon data-icon="inline-start" />
      Ready
    </Badge>
  )
}

function ResourceCard({
  item,
  onDetails,
  onCopy,
  copied,
}: {
  item: CatalogItem
  onDetails: (item: CatalogItem) => void
  onCopy: (item: CatalogItem) => void
  copied: boolean
}) {
  const Icon = item.icon

  return (
    <Card
      size="sm"
      className="resource-card h-full min-w-0 transition-transform duration-200 hover:-translate-y-0.5"
    >
      <CardHeader>
        <CardTitle>
          <span className="flex min-w-0 items-center gap-3">
            <span className="resource-icon" aria-hidden="true">
              <Icon />
            </span>
            <span className="truncate">{item.name}</span>
          </span>
        </CardTitle>
        <CardDescription className="truncate">
          {item.namespace}
        </CardDescription>
        <CardAction>
          <StatusBadge status={item.status} />
        </CardAction>
      </CardHeader>

      <CardContent className="flex min-h-24 flex-col gap-4">
        <p className="line-clamp-2 leading-5 text-muted-foreground">
          {item.summary}
        </p>
        <div className="mt-auto flex min-w-0 flex-col gap-2">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Globe2Icon className="size-3.5 shrink-0" aria-hidden="true" />
            <span className="truncate font-mono">{item.endpoint}</span>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <CableIcon className="size-3.5 shrink-0" aria-hidden="true" />
            <span className="truncate font-mono">
              {item.protocol} · {item.ports.join(", ")}
            </span>
          </div>
        </div>
      </CardContent>

      <CardFooter className="justify-between gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onDetails(item)}
        >
          <Settings2Icon data-icon="inline-start" />
          Details
        </Button>
        {item.href ? (
          <Button
            size="sm"
            render={
              <a
                href={item.href}
                target="_blank"
                rel="noopener noreferrer"
              />
            }
            nativeButton={false}
          >
            Open
            <ArrowUpRightIcon data-icon="inline-end" />
          </Button>
        ) : (
          <Button size="sm" variant="outline" onClick={() => onCopy(item)}>
            {copied ? (
              <CheckIcon data-icon="inline-start" />
            ) : (
              <CopyIcon data-icon="inline-start" />
            )}
            {copied ? "Copied" : "Copy address"}
          </Button>
        )}
      </CardFooter>
    </Card>
  )
}

export default function Home() {
  const searchRef = React.useRef<HTMLInputElement>(null)
  const [search, setSearch] = React.useState("")
  const [statusFilter, setStatusFilter] = React.useState("all")
  const [sortMode, setSortMode] = React.useState<SortMode>("recommended")
  const [showAllApps, setShowAllApps] = React.useState(false)
  const [showAllServices, setShowAllServices] = React.useState(false)
  const [selectedItem, setSelectedItem] = React.useState<CatalogItem | null>(null)
  const [copiedId, setCopiedId] = React.useState<string | null>(null)
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const [syncLabel, setSyncLabel] = React.useState(
    "Live snapshot · Jul 29, 2026"
  )

  React.useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault()
        searchRef.current?.focus()
      }

      if (event.key === "Escape" && document.activeElement === searchRef.current) {
        setSearch("")
        searchRef.current?.blur()
      }
    }

    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [])

  const filterCatalog = React.useCallback(
    (items: CatalogItem[]) => {
      const filtered = items.filter((item) => {
        const statusMatch =
          statusFilter === "all" || item.status === statusFilter
        return statusMatch && matchesSearch(item, search)
      })
      return sortItems(filtered, sortMode)
    },
    [search, sortMode, statusFilter]
  )

  const filteredApps = React.useMemo(
    () => filterCatalog(webApps),
    [filterCatalog]
  )
  const filteredServices = React.useMemo(
    () => filterCatalog(services),
    [filterCatalog]
  )

  const shouldShowAllApps = showAllApps || Boolean(search) || statusFilter !== "all"
  const shouldShowAllServices =
    showAllServices || Boolean(search) || statusFilter !== "all"
  const visibleApps = shouldShowAllApps ? filteredApps : filteredApps.slice(0, 8)
  const visibleServices = shouldShowAllServices
    ? filteredServices
    : filteredServices.slice(0, 8)
  const totalResults = filteredApps.length + filteredServices.length

  async function copyEndpoint(item: CatalogItem) {
    await navigator.clipboard.writeText(
      `${item.endpoint}:${item.ports[0]?.split(" ")[0]}`
    )
    setCopiedId(item.id)
    window.setTimeout(() => setCopiedId(null), 1600)
  }

  function refreshCatalog() {
    setIsRefreshing(true)
    window.setTimeout(() => {
      setIsRefreshing(false)
      setSyncLabel("Snapshot refreshed just now")
    }, 650)
  }

  function resetFilters() {
    setSearch("")
    setStatusFilter("all")
    setSortMode("recommended")
  }

  return (
    <main id="main-content" className="min-h-screen">
      <a className="skip-link" href="#catalog">
        Skip to catalog
      </a>

      <div className="mx-auto flex w-full max-w-[1480px] flex-col px-4 pb-16 sm:px-6 lg:px-8">
        <header className="flex flex-col gap-5 py-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex min-w-0 items-center gap-3">
            <span className="brand-mark" aria-hidden="true">
              <BoxesIcon />
            </span>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="truncate text-base font-semibold tracking-tight">
                  KubeDeck
                </span>
                <Badge variant="outline">Private</Badge>
              </div>
              <p className="truncate text-xs text-muted-foreground">
                rancher-desktop · v1.36.1+k3s1
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Dialog>
              <DialogTrigger
                render={<Button variant="outline" size="sm" />}
              >
                <span className="connection-dot" aria-hidden="true" />
                Cluster connected
              </DialogTrigger>
              <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                  <DialogTitle>Cluster connection</DialogTitle>
                  <DialogDescription>
                    The catalog captured from your Rancher Desktop Kubernetes
                    context.
                  </DialogDescription>
                </DialogHeader>
                <div className="flex flex-col gap-4">
                  <div className="detail-grid">
                    <span>Context</span>
                    <strong>rancher-desktop</strong>
                    <span>Kubernetes</span>
                    <strong>v1.36.1+k3s1</strong>
                    <span>Runtime</span>
                    <strong>containerd 2.3.2</strong>
                    <span>Discovery</span>
                    <strong>Ingress + Service</strong>
                    <span>Access mode</span>
                    <strong>Read-only catalog</strong>
                    <span>Snapshot</span>
                    <strong>Jul 29, 2026</strong>
                  </div>
                  <Separator />
                  <p className="text-sm leading-6 text-muted-foreground">
                    This private prototype uses a captured local catalog. A
                    lightweight in-cluster discovery agent is required for
                    continuous refresh without exposing your Kubernetes API.
                  </p>
                </div>
                <DialogFooter showCloseButton />
              </DialogContent>
            </Dialog>

            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={refreshCatalog}
                    aria-label="Refresh catalog snapshot"
                  />
                }
              >
                <RefreshCwIcon
                  className={cn(isRefreshing && "animate-spin")}
                />
              </TooltipTrigger>
              <TooltipContent>Refresh catalog snapshot</TooltipContent>
            </Tooltip>
          </div>
        </header>

        <section className="hero-panel" aria-labelledby="page-title">
          <div className="flex max-w-3xl flex-col gap-4">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="secondary">
                <CircleCheckIcon data-icon="inline-start" />
                1 node ready
              </Badge>
              <span className="text-xs text-muted-foreground">{syncLabel}</span>
            </div>
            <div>
              <h1
                id="page-title"
                className="max-w-2xl text-3xl font-semibold tracking-[-0.04em] text-balance sm:text-4xl"
              >
                Your Kubernetes ecosystem,
                <span className="text-primary"> one click away.</span>
              </h1>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground sm:text-base">
                A focused launchpad for every web UI and core service in your
                local cluster—discovered by namespace, route, and port.
              </p>
            </div>
          </div>

          <div className="resource-meter" aria-label="Node resource use">
            <div className="flex items-end justify-between gap-6">
              <div>
                <p className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">
                  Node pressure
                </p>
                <p className="mt-2 text-2xl font-semibold">Healthy</p>
              </div>
              <ActivityIcon className="size-7 text-primary" aria-hidden="true" />
            </div>
            <div className="mt-5 flex flex-col gap-4">
              <div>
                <div className="mb-2 flex justify-between text-xs">
                  <span className="text-muted-foreground">CPU</span>
                  <span className="font-mono">588m · 5%</span>
                </div>
                <div className="meter-track">
                  <span className="meter-fill w-[5%]" />
                </div>
              </div>
              <div>
                <div className="mb-2 flex justify-between text-xs">
                  <span className="text-muted-foreground">Memory</span>
                  <span className="font-mono">12.0 GiB · 38%</span>
                </div>
                <div className="meter-track">
                  <span className="meter-fill w-[38%]" />
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="stat-strip" aria-label="Cluster overview">
          {clusterStats.map((stat) => (
            <div key={stat.label} className="min-w-0">
              <p className="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
                {stat.label}
              </p>
              <p className="mt-2 text-xl font-semibold tracking-tight">
                {stat.value}
              </p>
              <p className="mt-1 truncate text-xs text-muted-foreground">
                {stat.detail}
              </p>
            </div>
          ))}
        </section>

        <section
          className="sticky-toolbar"
          aria-label="Catalog search and filters"
        >
          <div className="relative min-w-0 flex-1">
            <SearchIcon
              className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
              aria-hidden="true"
            />
            <Input
              ref={searchRef}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search apps, namespaces, URLs, or ports…"
              aria-label="Search Kubernetes catalog"
              className="h-10 pl-9 pr-14"
            />
            {search ? (
              <Button
                variant="ghost"
                size="icon-sm"
                className="absolute right-2 top-1/2 -translate-y-1/2"
                onClick={() => setSearch("")}
                aria-label="Clear search"
              >
                <XIcon />
              </Button>
            ) : (
              <kbd className="absolute right-3 top-1/2 hidden -translate-y-1/2 rounded-md border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground sm:inline-flex">
                ⌘ K
              </kbd>
            )}
          </div>

          <div className="flex items-center gap-2 overflow-x-auto pb-1 sm:pb-0">
            <ToggleGroup
              value={[statusFilter]}
              onValueChange={(value) => setStatusFilter(value[0] ?? "all")}
              variant="outline"
              size="sm"
              spacing={1}
              aria-label="Filter catalog by status"
            >
              <ToggleGroupItem value="all">All</ToggleGroupItem>
              <ToggleGroupItem value="ready">Ready</ToggleGroupItem>
              <ToggleGroupItem value="attention">Attention</ToggleGroupItem>
            </ToggleGroup>

            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="outline"
                    size="icon"
                    aria-label="Sort catalog"
                  />
                }
              >
                <ArrowDownAZIcon />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuGroup>
                  <DropdownMenuLabel>Sort catalog</DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => setSortMode("recommended")}
                  >
                    <SlidersHorizontalIcon />
                    Recommended
                    {sortMode === "recommended" && <CheckIcon className="ml-auto" />}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setSortMode("name")}>
                    <ArrowDownAZIcon />
                    Name
                    {sortMode === "name" && <CheckIcon className="ml-auto" />}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setSortMode("namespace")}>
                    <BoxesIcon />
                    Namespace
                    {sortMode === "namespace" && <CheckIcon className="ml-auto" />}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </section>

        <div id="catalog" className="flex flex-col gap-14 pt-10">
          {totalResults === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <SearchIcon />
                </EmptyMedia>
                <EmptyTitle>No cluster resources found</EmptyTitle>
                <EmptyDescription>
                  Try another name, namespace, URL, port, or status.
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button variant="outline" onClick={resetFilters}>
                  Reset filters
                </Button>
              </EmptyContent>
            </Empty>
          ) : (
            <>
              <section aria-labelledby="web-apps-title">
                <div className="section-heading">
                  <div>
                    <div className="flex items-center gap-2">
                      <Globe2Icon
                        className="size-4 text-primary"
                        aria-hidden="true"
                      />
                      <h2
                        id="web-apps-title"
                        className="text-lg font-semibold tracking-tight"
                      >
                        Web Apps
                      </h2>
                      <Badge variant="secondary">{filteredApps.length}</Badge>
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Browser-ready interfaces discovered from cluster ingresses.
                    </p>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    23 routes discovered
                  </p>
                </div>

                {filteredApps.length ? (
                  <>
                    <div className="catalog-grid">
                      {visibleApps.map((item) => (
                        <ResourceCard
                          key={item.id}
                          item={item}
                          onDetails={setSelectedItem}
                          onCopy={copyEndpoint}
                          copied={copiedId === item.id}
                        />
                      ))}
                    </div>
                    {!search &&
                      statusFilter === "all" &&
                      filteredApps.length > 8 && (
                        <div className="mt-5 flex justify-center">
                          <Button
                            variant="outline"
                            onClick={() => setShowAllApps((value) => !value)}
                          >
                            {showAllApps
                              ? "Show fewer apps"
                              : `Show all ${filteredApps.length} apps`}
                          </Button>
                        </div>
                      )}
                  </>
                ) : (
                  <p className="py-8 text-sm text-muted-foreground">
                    No web apps match this filter.
                  </p>
                )}
              </section>

              <section aria-labelledby="services-title">
                <div className="section-heading">
                  <div>
                    <div className="flex items-center gap-2">
                      <CableIcon
                        className="size-4 text-primary"
                        aria-hidden="true"
                      />
                      <h2
                        id="services-title"
                        className="text-lg font-semibold tracking-tight"
                      >
                        Services
                      </h2>
                      <Badge variant="secondary">
                        {filteredServices.length}
                      </Badge>
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Core APIs, data stores, brokers, and telemetry endpoints.
                    </p>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    66 services discovered
                  </p>
                </div>

                {filteredServices.length ? (
                  <>
                    <div className="catalog-grid">
                      {visibleServices.map((item) => (
                        <ResourceCard
                          key={item.id}
                          item={item}
                          onDetails={setSelectedItem}
                          onCopy={copyEndpoint}
                          copied={copiedId === item.id}
                        />
                      ))}
                    </div>
                    {!search &&
                      statusFilter === "all" &&
                      filteredServices.length > 8 && (
                        <div className="mt-5 flex justify-center">
                          <Button
                            variant="outline"
                            onClick={() =>
                              setShowAllServices((value) => !value)
                            }
                          >
                            {showAllServices
                              ? "Show fewer services"
                              : `Show all ${filteredServices.length} services`}
                          </Button>
                        </div>
                      )}
                  </>
                ) : (
                  <p className="py-8 text-sm text-muted-foreground">
                    No services match this filter.
                  </p>
                )}
              </section>
            </>
          )}
        </div>

        <footer className="mt-16 flex flex-col gap-3 border-t border-border py-7 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
          <p>KubeDeck · Read-only Kubernetes launchpad</p>
          <p>Ingress + Service discovery · Rancher Desktop</p>
        </footer>
      </div>

      <Dialog
        open={Boolean(selectedItem)}
        onOpenChange={(open) => {
          if (!open) setSelectedItem(null)
        }}
      >
        {selectedItem && (
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <div className="mb-1 flex items-center gap-3">
                <span className="resource-icon" aria-hidden="true">
                  <selectedItem.icon />
                </span>
                <StatusBadge status={selectedItem.status} />
              </div>
              <DialogTitle>{selectedItem.name}</DialogTitle>
              <DialogDescription>{selectedItem.summary}</DialogDescription>
            </DialogHeader>

            <div className="flex flex-col gap-4">
              <div className="detail-grid">
                <span>Namespace</span>
                <strong>{selectedItem.namespace}</strong>
                <span>Endpoint</span>
                <strong className="truncate font-mono">
                  {selectedItem.endpoint}
                </strong>
                <span>Protocol</span>
                <strong>{selectedItem.protocol}</strong>
                <span>Ports</span>
                <strong className="font-mono">
                  {selectedItem.ports.join(", ")}
                </strong>
                <span>Workload</span>
                <strong>{selectedItem.workload}</strong>
                <span>Pods</span>
                <strong>{selectedItem.pods}</strong>
                <span>Discovered from</span>
                <strong>{selectedItem.source}</strong>
              </div>
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => copyEndpoint(selectedItem)}
              >
                {copiedId === selectedItem.id ? (
                  <CheckIcon data-icon="inline-start" />
                ) : (
                  <CopyIcon data-icon="inline-start" />
                )}
                {copiedId === selectedItem.id ? "Copied" : "Copy address"}
              </Button>
              {selectedItem.href && (
                <Button
                  render={
                    <a
                      href={selectedItem.href}
                      target="_blank"
                      rel="noopener noreferrer"
                    />
                  }
                  nativeButton={false}
                >
                  Open {selectedItem.name}
                  <ExternalLinkIcon data-icon="inline-end" />
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </main>
  )
}
