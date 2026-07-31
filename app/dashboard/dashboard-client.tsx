"use client"

import * as React from "react"
import type { LucideIcon } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"
import {
  CartesianGrid,
  Line,
  LineChart,
  XAxis,
  YAxis,
} from "recharts"
import {
  ActivityIcon,
  AppWindowIcon,
  ArrowDownAZIcon,
  ArrowUpRightIcon,
  BarChart3Icon,
  BotIcon,
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
  CpuIcon,
  DatabaseIcon,
  ExternalLinkIcon,
  EyeIcon,
  GitBranchIcon,
  Globe2Icon,
  HardDriveIcon,
  InboxIcon,
  KeyRoundIcon,
  Layers3Icon,
  LogOutIcon,
  MailIcon,
  MemoryStickIcon,
  MessageSquareIcon,
  NetworkIcon,
  RocketIcon,
  SearchIcon,
  ServerIcon,
  ServerCogIcon,
  Settings2Icon,
  ShieldCheckIcon,
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
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
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
import {
  Progress,
  ProgressLabel,
  ProgressValue,
} from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Radio,
  RadioGroup,
  RadioIndicator,
} from "@/components/animate-ui/primitives/base/radio"
import { KubeDeckBanner } from "@/components/kubedeck-banner"
import { KubeDeckLogo } from "@/components/kubedeck-logo"
import { NotificationsMenu } from "@/components/notifications-menu"
import {
  getNodeResourceTrend,
  monitoringSnapshot,
  nodeSnapshots,
  nodeSummary,
  type NodeSnapshot,
} from "@/lib/kubedeck-monitoring"
import { cn } from "@/lib/utils"

type CatalogStatus = "ready" | "attention"
type CatalogKind = "app" | "service"
type KindFilter = CatalogKind | "all"
type StatusFilter = CatalogStatus | "all"
type SortMode = "recommended" | "name" | "namespace"
export type DashboardView = "overview" | "nodes" | "dns" | "catalog"
type DashboardAdmin = {
  firstName: string
  lastName: string
  email: string
  role: "admin"
}
type CategoryId =
  | "web-applications"
  | "databases-storage"
  | "observability-metrics"
  | "automation-workflows"
  | "deployments"
  | "ai-services"
  | "messaging-events"
  | "developer-tools"
  | "platform-security"

type CatalogBase = {
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

type OperationalMeta = {
  category: CategoryId
  serviceName: string
  clusterIP: string
  readyEndpoints: number
  totalEndpoints: number
  uptime: string
  uptimeHours: number
  lastDeployedAt: string
}

type CatalogItem = CatalogBase &
  OperationalMeta & {
    externalDomain?: string
    internalDns: string
    readiness: number
  }

const webApps: CatalogBase[] = [
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

const services: CatalogBase[] = [
  {
    id: "homelab-api",
    name: "Homelab API",
    kind: "service",
    namespace: "apps",
    summary: "Example application API and operational endpoints.",
    endpoint: "api.dev.local",
    href: "https://api.dev.local",
    ports: ["3000"],
    protocol: "HTTPS",
    workload: "Deployment/homelab-api",
    pods: "2 / 2 ready",
    source: "Ingress/homelab-api",
    status: "ready",
    icon: SparklesIcon,
  },
  {
    id: "mcp-gateway",
    name: "MCP Gateway",
    kind: "service",
    namespace: "ai-services",
    summary: "Shared Model Context Protocol gateway for local tools.",
    endpoint: "mcp-gateway.dev.local",
    href: "https://mcp-gateway.dev.local",
    ports: ["3000"],
    protocol: "HTTPS",
    workload: "Deployment/mcp-gateway",
    pods: "1 / 1 ready",
    source: "Ingress/mcp-gateway",
    status: "ready",
    icon: CloudCogIcon,
  },
  {
    id: "model-runner",
    name: "Model Runner",
    kind: "service",
    namespace: "ai-services",
    summary: "Example inference and model-serving endpoint.",
    endpoint: "model-runner.dev.local",
    href: "https://model-runner.dev.local",
    ports: ["3000"],
    protocol: "HTTPS",
    workload: "Deployment/model-runner",
    pods: "1 / 1 ready",
    source: "Ingress/model-runner",
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

const clusterDomain = "cluster.local"
const capturedAt = monitoringSnapshot.capturedAt

const operationalMeta: Record<string, OperationalMeta> = {
  grafana: {
    category: "observability-metrics",
    serviceName: "grafana",
    clusterIP: "10.43.1.72",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "4d 9h",
    uptimeHours: 105.97,
    lastDeployedAt: "2026-07-24T19:07:45Z",
  },
  radar: {
    category: "observability-metrics",
    serviceName: "radar",
    clusterIP: "10.43.24.183",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "1h 2m",
    uptimeHours: 1.04,
    lastDeployedAt: "2026-07-29T04:03:48Z",
  },
  infisical: {
    category: "platform-security",
    serviceName: "infisical-backend",
    clusterIP: "10.43.11.43",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "58m",
    uptimeHours: 0.98,
    lastDeployedAt: "2026-07-29T04:07:02Z",
  },
  n8n: {
    category: "automation-workflows",
    serviceName: "n8n",
    clusterIP: "10.43.56.192",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 12h",
    uptimeHours: 252.93,
    lastDeployedAt: "2026-07-18T16:10:06Z",
  },
  temporal: {
    category: "automation-workflows",
    serviceName: "temporal-web",
    clusterIP: "10.43.151.67",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.4,
    lastDeployedAt: "2026-07-18T08:41:41Z",
  },
  "nats-ui": {
    category: "messaging-events",
    serviceName: "nats-ui",
    clusterIP: "10.43.71.175",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.51,
    lastDeployedAt: "2026-07-18T08:35:14Z",
  },
  "kafka-ui": {
    category: "messaging-events",
    serviceName: "kafka-ui",
    clusterIP: "10.43.47.96",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.68,
    lastDeployedAt: "2026-07-18T08:25:06Z",
  },
  "minio-console": {
    category: "databases-storage",
    serviceName: "minio",
    clusterIP: "10.43.121.253",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.7,
    lastDeployedAt: "2026-07-18T08:24:10Z",
  },
  pgadmin: {
    category: "databases-storage",
    serviceName: "pgadmin",
    clusterIP: "10.43.143.243",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.49,
    lastDeployedAt: "2026-07-18T08:36:18Z",
  },
  "redis-commander": {
    category: "databases-storage",
    serviceName: "redis-commander",
    clusterIP: "10.43.27.142",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.49,
    lastDeployedAt: "2026-07-18T08:36:50Z",
  },
  rabbitmq: {
    category: "messaging-events",
    serviceName: "rabbitmq",
    clusterIP: "10.43.66.194",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.72,
    lastDeployedAt: "2026-07-18T08:22:36Z",
  },
  mailpit: {
    category: "developer-tools",
    serviceName: "mailpit-ui",
    clusterIP: "10.43.202.31",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.5,
    lastDeployedAt: "2026-07-18T08:36:06Z",
  },
  "mongo-ui": {
    category: "databases-storage",
    serviceName: "mongoku",
    clusterIP: "10.43.35.66",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.67,
    lastDeployedAt: "2026-07-18T08:25:49Z",
  },
  "schema-registry-ui": {
    category: "messaging-events",
    serviceName: "schema-registry-ui",
    clusterIP: "10.43.233.88",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.47,
    lastDeployedAt: "2026-07-18T08:37:41Z",
  },
  "jaeger-ui": {
    category: "observability-metrics",
    serviceName: "jaeger-ui",
    clusterIP: "10.43.95.112",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.68,
    lastDeployedAt: "2026-07-18T08:24:55Z",
  },
  grpcui: {
    category: "web-applications",
    serviceName: "grpcui",
    clusterIP: "10.43.19.161",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.69,
    lastDeployedAt: "2026-07-18T08:24:44Z",
  },
  "k6-dashboard": {
    category: "deployments",
    serviceName: "k6-web-dashboard",
    clusterIP: "10.43.4.252",
    readyEndpoints: 0,
    totalEndpoints: 0,
    uptime: "Not running",
    uptimeHours: 0,
    lastDeployedAt: "2026-07-16T18:16:20Z",
  },
  "homelab-api": {
    category: "web-applications",
    serviceName: "homelab-api",
    clusterIP: "10.43.99.211",
    readyEndpoints: 2,
    totalEndpoints: 2,
    uptime: "7h 48m",
    uptimeHours: 7.82,
    lastDeployedAt: "2026-07-28T21:17:19Z",
  },
  "mcp-gateway": {
    category: "ai-services",
    serviceName: "mcp-gateway",
    clusterIP: "10.43.188.213",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "4d 22h",
    uptimeHours: 118.24,
    lastDeployedAt: "2026-07-24T06:51:20Z",
  },
  "model-runner": {
    category: "ai-services",
    serviceName: "model-runner",
    clusterIP: "10.43.200.69",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "1d 9h",
    uptimeHours: 33.94,
    lastDeployedAt: "2026-07-27T19:09:47Z",
  },
  postgresql: {
    category: "databases-storage",
    serviceName: "postgresql",
    clusterIP: "10.43.19.66",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "4d 17h",
    uptimeHours: 113.69,
    lastDeployedAt: "2026-07-24T11:24:21Z",
  },
  redis: {
    category: "databases-storage",
    serviceName: "redis",
    clusterIP: "10.43.251.8",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.73,
    lastDeployedAt: "2026-07-18T08:22:23Z",
  },
  mongodb: {
    category: "databases-storage",
    serviceName: "mongodb",
    clusterIP: "10.43.255.200",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.71,
    lastDeployedAt: "2026-07-18T08:23:33Z",
  },
  nats: {
    category: "messaging-events",
    serviceName: "nats",
    clusterIP: "10.43.233.25",
    readyEndpoints: 3,
    totalEndpoints: 3,
    uptime: "1d 19h",
    uptimeHours: 43.28,
    lastDeployedAt: "2026-07-27T09:49:43Z",
  },
  kafka: {
    category: "messaging-events",
    serviceName: "kafka",
    clusterIP: "10.43.114.51",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.72,
    lastDeployedAt: "2026-07-18T08:23:00Z",
  },
  prometheus: {
    category: "observability-metrics",
    serviceName: "monitoring-kube-prometheus-prometheus",
    clusterIP: "10.43.216.124",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "4d 9h",
    uptimeHours: 105.97,
    lastDeployedAt: "2026-07-24T19:07:46Z",
  },
  loki: {
    category: "observability-metrics",
    serviceName: "loki",
    clusterIP: "10.43.140.26",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "9h 57m",
    uptimeHours: 9.95,
    lastDeployedAt: "2026-07-28T19:08:55Z",
  },
  tempo: {
    category: "observability-metrics",
    serviceName: "tempo",
    clusterIP: "10.43.236.243",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "4d 9h",
    uptimeHours: 105.96,
    lastDeployedAt: "2026-07-24T19:08:23Z",
  },
  alloy: {
    category: "observability-metrics",
    serviceName: "alloy",
    clusterIP: "10.43.123.64",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "9h 57m",
    uptimeHours: 9.96,
    lastDeployedAt: "2026-07-28T19:08:33Z",
  },
  "temporal-frontend": {
    category: "automation-workflows",
    serviceName: "temporal-frontend",
    clusterIP: "10.43.73.124",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.41,
    lastDeployedAt: "2026-07-18T08:41:39Z",
  },
  "minio-api": {
    category: "databases-storage",
    serviceName: "minio",
    clusterIP: "10.43.121.253",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.7,
    lastDeployedAt: "2026-07-18T08:24:10Z",
  },
  "schema-registry": {
    category: "messaging-events",
    serviceName: "schema-registry",
    clusterIP: "10.43.207.216",
    readyEndpoints: 1,
    totalEndpoints: 1,
    uptime: "10d 20h",
    uptimeHours: 260.48,
    lastDeployedAt: "2026-07-18T08:37:10Z",
  },
}

const categoryConfig: {
  id: CategoryId
  label: string
  description: string
  icon: LucideIcon
}[] = [
  {
    id: "web-applications",
    label: "Web Applications",
    description: "Launchable application and API surfaces with external URLs.",
    icon: AppWindowIcon,
  },
  {
    id: "databases-storage",
    label: "Databases & Storage",
    description: "Persistent data, caches, object storage, and admin tools.",
    icon: DatabaseIcon,
  },
  {
    id: "observability-metrics",
    label: "Observability & Metrics",
    description: "Metrics, logs, traces, topology, and telemetry transport.",
    icon: ActivityIcon,
  },
  {
    id: "automation-workflows",
    label: "Automation & Workflows",
    description: "Workflow engines, orchestration, and integration surfaces.",
    icon: WorkflowIcon,
  },
  {
    id: "deployments",
    label: "Deployments & Testing",
    description: "Release surfaces, ephemeral workloads, and load-test runs.",
    icon: RocketIcon,
  },
  {
    id: "ai-services",
    label: "AI & MCP Services",
    description: "AI-facing MCP endpoints, agents, and engineering services.",
    icon: BotIcon,
  },
  {
    id: "messaging-events",
    label: "Messaging & Events",
    description: "Brokers, streams, schemas, and event administration.",
    icon: MessageSquareIcon,
  },
  {
    id: "developer-tools",
    label: "Developer Tools",
    description: "Email testing, protocol exploration, and engineering tools.",
    icon: CloudCogIcon,
  },
  {
    id: "platform-security",
    label: "Platform & Security",
    description: "Secrets, access control, and core platform capabilities.",
    icon: ShieldCheckIcon,
  },
]

const catalogItems: CatalogItem[] = [...webApps, ...services].map((item) => {
  const meta = operationalMeta[item.id]
  const internalDns = `${meta.serviceName}.${item.namespace}.svc.${clusterDomain}`
  const readiness =
    meta.totalEndpoints > 0
      ? Math.round((meta.readyEndpoints / meta.totalEndpoints) * 100)
      : 0

  return {
    ...item,
    ...meta,
    externalDomain: item.href ? item.endpoint : undefined,
    internalDns,
    readiness,
  }
})

const clusterStats = [
  { label: "Clusters", value: "All", detail: "registered contexts" },
  {
    label: "Nodes",
    value: String(nodeSummary.total),
    detail: `${nodeSummary.ready} ready · ${nodeSummary.attention} attention`,
  },
  { label: "Namespaces", value: "18", detail: "active cluster scopes" },
  { label: "Workloads", value: "46", detail: "deployments + stateful sets" },
  {
    label: "Pods",
    value: `${nodeSummary.pods} / ${nodeSummary.podCapacity}`,
    detail: "allocated across sample nodes",
  },
] as const

const nodeResources: {
  label: string
  value: string
  detail: string
  usage: number
  icon: LucideIcon
}[] = [
  {
    label: "CPU",
    value: "48%",
    detail: "fleet average",
    usage: 48,
    icon: CpuIcon,
  },
  {
    label: "Memory",
    value: "65%",
    detail: "fleet average",
    usage: 65,
    icon: MemoryStickIcon,
  },
  {
    label: "Storage",
    value: "52%",
    detail: "fleet average",
    usage: 52,
    icon: HardDriveIcon,
  },
  {
    label: "Pod allocation",
    value: "69%",
    detail: `${nodeSummary.pods} / ${nodeSummary.podCapacity} pod capacity`,
    usage: 69,
    icon: BoxesIcon,
  },
]

const nodeChartConfig = {
  cpu: {
    label: "CPU",
    color: "var(--chart-1)",
  },
  memory: {
    label: "Memory",
    color: "var(--chart-2)",
  },
  storage: {
    label: "Storage",
    color: "var(--chart-3)",
  },
  pods: {
    label: "Pod allocation",
    color: "var(--chart-4)",
  },
} satisfies ChartConfig

function getCategory(category: CategoryId) {
  return categoryConfig.find((item) => item.id === category)!
}

function formatUtcTimestamp(timestamp: string) {
  const date = new Date(timestamp)
  const months = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ]
  const hours = String(date.getUTCHours()).padStart(2, "0")
  const minutes = String(date.getUTCMinutes()).padStart(2, "0")

  return `${months[date.getUTCMonth()]} ${date.getUTCDate()}, ${date.getUTCFullYear()} · ${hours}:${minutes} UTC`
}

function matchesSearch(item: CatalogItem, search: string) {
  const needle = search.trim().toLowerCase()
  if (!needle) return true

  return [
    item.name,
    item.namespace,
    item.summary,
    item.externalDomain,
    item.internalDns,
    item.clusterIP,
    item.serviceName,
    item.protocol,
    item.ports.join(" "),
    getCategory(item.category).label,
  ]
    .filter(Boolean)
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

function ResourceMetric({
  label,
  value,
  detail,
  usage,
  icon: Icon,
}: (typeof nodeResources)[number]) {
  return (
    <div className="resource-metric">
      <div className="flex items-center justify-between gap-3">
        <span className="flex min-w-0 items-center gap-2 text-xs font-medium text-muted-foreground">
          <Icon className="size-3.5 shrink-0 text-primary" aria-hidden="true" />
          <span className="truncate">{label}</span>
        </span>
        <strong className="shrink-0 text-sm font-semibold tabular-nums">
          {value}
        </strong>
      </div>
      <Progress
        value={usage}
        aria-label={`${label}: ${value}`}
        className="mt-3"
      />
      <p className="mt-2 truncate text-[10px] text-muted-foreground">
        {detail}
      </p>
    </div>
  )
}

function NodeStatusBadge({ node }: { node: NodeSnapshot }) {
  if (node.status === "attention") {
    return (
      <Badge variant="destructive">
        <CircleDotDashedIcon data-icon="inline-start" />
        Attention
      </Badge>
    )
  }

  return (
    <Badge variant="secondary">
      <CircleCheckIcon data-icon="inline-start" />
      Ready
    </Badge>
  )
}

function MultiNodeReview({
  showControlPlaneNodes,
  showWorkerNodes,
  highlightNodePressure,
}: {
  showControlPlaneNodes: boolean
  showWorkerNodes: boolean
  highlightNodePressure: boolean
}) {
  const [selectedNodeId, setSelectedNodeId] = React.useState("fleet")
  const visibleNodes = nodeSnapshots.filter(
    (node) =>
      (node.role === "control-plane" && showControlPlaneNodes) ||
      (node.role === "worker" && showWorkerNodes)
  )
  const activeNodeId =
    selectedNodeId === "fleet" ||
    visibleNodes.some((node) => node.id === selectedNodeId)
      ? selectedNodeId
      : "fleet"
  const activeNode = nodeSnapshots.find((node) => node.id === activeNodeId)
  const chartData = getNodeResourceTrend(activeNodeId)
  const activeLabel = activeNode?.name ?? "Fleet average"

  return (
    <section
      id="node-review"
      className="node-review"
      aria-labelledby="node-review-title"
    >
      <div className="node-review-heading">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="secondary">
              <ServerIcon data-icon="inline-start" />
              Multi-node
            </Badge>
            <Badge variant="outline">{monitoringSnapshot.mode}</Badge>
          </div>
          <h2 id="node-review-title">Cluster node review</h2>
          <p>
            Compare node conditions and four normalized resource signals across
            the selected Kubernetes scope.
          </p>
        </div>
        <RadioGroup
          value={activeNodeId}
          onValueChange={setSelectedNodeId}
          aria-label="Select node telemetry scope"
          className="node-scope-selector"
        >
          <Radio value="fleet">
            <RadioIndicator />
            Fleet
          </Radio>
          {visibleNodes.map((node) => (
            <Radio key={node.id} value={node.id}>
              <RadioIndicator />
              {node.name}
            </Radio>
          ))}
        </RadioGroup>
      </div>

      <div className="node-review-grid">
        <Card className="node-chart-card">
          <CardHeader>
            <CardTitle>Resource history</CardTitle>
            <CardDescription>
              {activeLabel} · normalized percentage over the preview window
            </CardDescription>
            <CardAction>
              <Badge variant="outline">4 properties</Badge>
            </CardAction>
          </CardHeader>
          <CardContent>
            <ChartContainer
              config={nodeChartConfig}
              className="node-resource-chart"
              initialDimension={{ width: 760, height: 292 }}
              aria-label={`${activeLabel} CPU, memory, storage, and pod allocation line chart`}
            >
              <LineChart
                accessibilityLayer
                data={chartData}
                margin={{ left: 4, right: 12, top: 8 }}
              >
                <CartesianGrid vertical={false} strokeDasharray="3 5" />
                <XAxis
                  dataKey="time"
                  tickLine={false}
                  axisLine={false}
                  tickMargin={10}
                />
                <YAxis
                  domain={[0, 100]}
                  width={38}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value) => `${value}%`}
                />
                <ChartTooltip
                  cursor={false}
                  content={<ChartTooltipContent indicator="line" />}
                />
                <ChartLegend content={<ChartLegendContent />} />
                {Object.keys(nodeChartConfig).map((property) => (
                  <Line
                    key={property}
                    dataKey={property}
                    type="monotone"
                    stroke={`var(--color-${property})`}
                    strokeWidth={2.2}
                    dot={false}
                    activeDot={{ r: 4 }}
                  />
                ))}
              </LineChart>
            </ChartContainer>
          </CardContent>
          <CardFooter className="justify-between gap-3">
            <span>
              {monitoringSnapshot.source} · captured {capturedAt}
            </span>
            <Badge variant="outline">Preview data</Badge>
          </CardFooter>
        </Card>

        <Card className="node-table-card">
          <CardHeader>
            <CardTitle>Node status</CardTitle>
            <CardDescription>
              {visibleNodes.length} visible nodes · {nodeSummary.attention} need
              attention
            </CardDescription>
          </CardHeader>
          <CardContent>
            {visibleNodes.length > 0 ? (
              <Table>
                <TableCaption className="sr-only">
                  Multi-node Kubernetes status snapshot
                </TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>Node</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>CPU</TableHead>
                    <TableHead>Memory</TableHead>
                    <TableHead>Storage</TableHead>
                    <TableHead>Pods</TableHead>
                    <TableHead>Heartbeat</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visibleNodes.map((node) => (
                    <TableRow
                      key={node.id}
                      className={cn(
                        highlightNodePressure &&
                          node.status === "attention" &&
                          "node-row--attention"
                      )}
                    >
                      <TableCell>
                        <span className="node-name-cell">
                          <strong>{node.name}</strong>
                          <small>
                            {node.cluster} ·{" "}
                            {node.role === "control-plane"
                              ? "control plane"
                              : "worker"}{" "}
                            · {node.internalIp}
                          </small>
                        </span>
                      </TableCell>
                      <TableCell>
                        <NodeStatusBadge node={node} />
                        <small className="node-condition">
                          {node.condition}
                        </small>
                      </TableCell>
                      <TableCell>{node.cpu}%</TableCell>
                      <TableCell>{node.memory}%</TableCell>
                      <TableCell>{node.storage}%</TableCell>
                      <TableCell>{node.pods}</TableCell>
                      <TableCell>
                        <span className="node-heartbeat">
                          <strong>{node.heartbeat}</strong>
                          <small>
                            {node.kubeletVersion} · {node.age}
                          </small>
                        </span>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <ServerIcon />
                  </EmptyMedia>
                  <EmptyTitle>No node roles selected</EmptyTitle>
                  <EmptyDescription>
                    Enable control-plane or worker nodes in Kubernetes
                    Settings.
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            )}
          </CardContent>
        </Card>
      </div>
    </section>
  )
}

function AvailabilityGraphic({ item }: { item: CatalogItem }) {
  const endpointLabel =
    item.totalEndpoints > 0
      ? `${item.readyEndpoints}/${item.totalEndpoints}`
      : "0"

  return (
    <div className="availability-panel">
      <div
        className={cn(
          "status-ring",
          item.status === "attention" && "is-attention"
        )}
        role="img"
        aria-label={`${item.name}: ${endpointLabel} endpoints ready`}
      >
        <span>{endpointLabel}</span>
        <small>ready</small>
      </div>
      <div className="min-w-0 flex-1">
        <Progress
          value={Math.min((item.uptimeHours / 168) * 100, 100)}
          aria-label={`${item.name} current run age: ${item.uptime}`}
          className="gap-2"
        >
          <ProgressLabel>
            {item.status === "attention" ? "Run state" : "Current run"}
          </ProgressLabel>
          <ProgressValue>{() => item.uptime}</ProgressValue>
        </Progress>
        <div className="mt-2 flex min-w-0 items-center gap-1.5 text-[11px] text-muted-foreground">
          <span className="shrink-0 font-medium text-foreground/80">
            Last deploy
          </span>
          <span aria-hidden="true">·</span>
          <time
            dateTime={item.lastDeployedAt}
            className="truncate tabular-nums"
            title={item.lastDeployedAt}
          >
            {formatUtcTimestamp(item.lastDeployedAt)}
          </time>
        </div>
        <p className="mt-1 text-[10px] text-muted-foreground">
          {item.status === "attention"
            ? "No active endpoint at capture"
            : "Current run shown on a 7-day scale"}
        </p>
      </div>
    </div>
  )
}

function ResourceCard({
  item,
  onDetails,
  onCopy,
  copied,
  index,
  openInNewTab,
}: {
  item: CatalogItem
  onDetails: (item: CatalogItem) => void
  onCopy: (item: CatalogItem) => void
  copied: boolean
  index: number
  openInNewTab: boolean
}) {
  const Icon = item.icon

  return (
    <motion.div
      layout
      className="resource-card-motion h-full min-w-0"
      initial={{ opacity: 0, scale: 0.975, y: 18 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.975, y: -10 }}
      whileHover={{ y: -4 }}
      transition={{
        type: "spring",
        stiffness: 280,
        damping: 26,
        delay: Math.min(index * 0.025, 0.18),
      }}
    >
      <Card size="sm" className="resource-card h-full min-w-0">
        <CardHeader>
          <CardTitle>
            <span className="flex min-w-0 items-center gap-3">
              <span className="resource-icon" aria-hidden="true">
                <Icon />
              </span>
              <span className="truncate">{item.name}</span>
            </span>
          </CardTitle>
          <CardDescription className="flex min-w-0 items-center gap-2">
            <span className="truncate">{item.namespace}</span>
            <span aria-hidden="true">·</span>
            <span className="shrink-0">
              {item.kind === "app" ? "Web app" : "Service"}
            </span>
          </CardDescription>
          <CardAction>
            <StatusBadge status={item.status} />
          </CardAction>
        </CardHeader>

        <CardContent className="flex min-h-64 flex-col gap-4">
          <p className="line-clamp-2 leading-5 text-muted-foreground">
            {item.summary}
          </p>

          <AvailabilityGraphic item={item} />

          <div className="mt-auto flex min-w-0 flex-col gap-2.5">
            {item.externalDomain && (
              <div className="address-line">
                <Globe2Icon aria-hidden="true" />
                <span>
                  <small>Ingress</small>
                  <strong>{item.externalDomain}</strong>
                </span>
              </div>
            )}
            <div className="address-line">
              <NetworkIcon aria-hidden="true" />
              <span>
                <small>Cluster DNS</small>
                <strong>{item.internalDns}</strong>
              </span>
            </div>
            <div className="address-line">
              <CableIcon aria-hidden="true" />
              <span>
                <small>Ports</small>
                <strong>
                  {item.protocol} · {item.ports.join(", ")}
                </strong>
              </span>
            </div>
          </div>
        </CardContent>

        <CardFooter className="justify-between gap-2">
          <Button variant="ghost" size="sm" onClick={() => onDetails(item)}>
            <Settings2Icon data-icon="inline-start" />
            Details
          </Button>
          {item.href ? (
            <Button
              size="sm"
              render={
                <a
                  href={item.href}
                  target={openInNewTab ? "_blank" : undefined}
                  rel={openInNewTab ? "noopener noreferrer" : undefined}
                />
              }
              nativeButton={false}
            >
              Open
              <ArrowUpRightIcon data-icon="inline-end" />
            </Button>
          ) : (
            <Button size="sm" variant="outline" onClick={() => onCopy(item)}>
              <AnimatePresence mode="wait" initial={false}>
                <motion.span
                  key={copied ? "copied" : "copy"}
                  className="inline-flex items-center"
                  initial={{ opacity: 0, scale: 0.7 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.7 }}
                >
                  {copied ? (
                    <CheckIcon data-icon="inline-start" />
                  ) : (
                    <CopyIcon data-icon="inline-start" />
                  )}
                </motion.span>
              </AnimatePresence>
              {copied ? "Copied" : "Copy DNS"}
            </Button>
          )}
        </CardFooter>
      </Card>
    </motion.div>
  )
}

const dashboardViewMeta: Record<
  DashboardView,
  { eyebrow: string; title: string; description: string }
> = {
  overview: {
    eyebrow: "Fleet overview",
    title: "Kubernetes operations dashboard",
    description:
      "Monitor the global cluster scope, review resource pressure, and move directly to the operational workspace you need.",
  },
  nodes: {
    eyebrow: "Cluster infrastructure",
    title: "Nodes and resource health",
    description:
      "Compare Kubernetes nodes, pressure conditions, heartbeat state, and four normalized resource signals.",
  },
  dns: {
    eyebrow: "Cluster networking",
    title: "DNS and service discovery",
    description:
      "Review the active cluster domain, CoreDNS service identity, resolver IP, and pod search path.",
  },
  catalog: {
    eyebrow: "Service catalog",
    title: "Applications and services",
    description:
      "Browse one operational category at a time with focused search, readiness, DNS, ports, uptime, and deployment details.",
  },
}

export default function DashboardClient({
  admin,
  view = "overview",
  catalogCategory,
}: {
  admin: DashboardAdmin
  view?: DashboardView
  catalogCategory?: string
}) {
  const searchRef = React.useRef<HTMLInputElement>(null)
  const [search, setSearch] = React.useState("")
  const [kindFilter, setKindFilter] = React.useState<KindFilter>("all")
  const [statusFilter, setStatusFilter] = React.useState<StatusFilter>("all")
  const [sortMode, setSortMode] = React.useState<SortMode>("recommended")
  const [selectedItem, setSelectedItem] = React.useState<CatalogItem | null>(null)
  const [copiedId, setCopiedId] = React.useState<string | null>(null)
  const [compactCatalog, setCompactCatalog] = React.useState(false)
  const [openInNewTab, setOpenInNewTab] = React.useState(true)
  const [showControlPlaneNodes, setShowControlPlaneNodes] =
    React.useState(true)
  const [showWorkerNodes, setShowWorkerNodes] = React.useState(true)
  const [highlightNodePressure, setHighlightNodePressure] =
    React.useState(true)

  React.useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (view !== "catalog") return

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
  }, [view])

  React.useEffect(() => {
    function readPreferences() {
      setCompactCatalog(
        window.localStorage.getItem("kubedeck-compact-catalog") === "true"
      )
      setOpenInNewTab(
        window.localStorage.getItem("kubedeck-external-links") !== "false"
      )
      setShowControlPlaneNodes(
        window.localStorage.getItem("kubedeck-show-control-plane") !== "false"
      )
      setShowWorkerNodes(
        window.localStorage.getItem("kubedeck-show-workers") !== "false"
      )
      setHighlightNodePressure(
        window.localStorage.getItem("kubedeck-highlight-node-pressure") !==
          "false"
      )
    }

    readPreferences()
    window.addEventListener("kubedeck:preferences", readPreferences)
    return () =>
      window.removeEventListener("kubedeck:preferences", readPreferences)
  }, [])

  const activeCategory =
    categoryConfig.find((category) => category.id === catalogCategory) ??
    categoryConfig[0]

  const filteredItems = React.useMemo(() => {
    const filtered = catalogItems.filter((item) => {
      const categoryMatch =
        view !== "catalog" || item.category === activeCategory.id
      const kindMatch = kindFilter === "all" || item.kind === kindFilter
      const statusMatch =
        statusFilter === "all" || item.status === statusFilter
      return (
        categoryMatch &&
        kindMatch &&
        statusMatch &&
        matchesSearch(item, search)
      )
    })

    return sortItems(filtered, sortMode)
  }, [
    activeCategory.id,
    kindFilter,
    search,
    sortMode,
    statusFilter,
    view,
  ])

  const groupedCategories = categoryConfig
    .map((category) => ({
      ...category,
      items: filteredItems.filter((item) => item.category === category.id),
    }))
    .filter((category) => category.items.length > 0)

  async function copyClusterDns(item: CatalogItem) {
    await navigator.clipboard.writeText(
      `${item.internalDns}:${item.ports[0]?.split(" ")[0]}`
    )
    setCopiedId(item.id)
    window.setTimeout(() => setCopiedId(null), 1600)
  }

  function resetFilters() {
    setSearch("")
    setKindFilter("all")
    setStatusFilter("all")
    setSortMode("recommended")
  }

  const adminInitials =
    `${admin.firstName.at(0) ?? ""}${admin.lastName.at(0) ?? ""}`.toUpperCase()

  return (
    <main
      id="main-content"
      className={cn(
        "dashboard-shell liquid-stage",
        compactCatalog && "dashboard-shell--compact"
      )}
    >
      <a className="skip-link" href="#dashboard-page-content">
        Skip to page content
      </a>

      <div className="liquid-grid" aria-hidden="true" />
      <div
        className="liquid-orbit liquid-orbit--dashboard"
        aria-hidden="true"
      />

      <aside className="dashboard-sidebar" aria-label="Primary navigation">
        <a
          className="sidebar-brand"
          href="/dashboard"
          aria-label="KubeDeck home"
        >
          <KubeDeckLogo className="brand-mark" priority />
          <span className="sr-only">KubeDeck</span>
        </a>
        <nav className="dashboard-nav">
          <a
            className={cn(view === "overview" && "is-active")}
            href="/dashboard"
            aria-current={view === "overview" ? "page" : undefined}
            aria-label="Overview"
            data-label="Overview"
          >
            <CircleGaugeIcon />
          </a>
          <a
            className={cn(view === "nodes" && "is-active")}
            href="/dashboard/nodes"
            aria-current={view === "nodes" ? "page" : undefined}
            aria-label="Cluster node review"
            data-label="Nodes"
          >
            <ServerCogIcon />
          </a>
          <a
            className={cn(view === "dns" && "is-active")}
            href="/dashboard/dns"
            aria-current={view === "dns" ? "page" : undefined}
            aria-label="Cluster DNS"
            data-label="DNS"
          >
            <NetworkIcon />
          </a>
          {/* Standard links avoid vinext's unsupported dynamic router import. */}
          {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
          <a
            className={cn(
              view === "catalog" &&
                activeCategory.id === "databases-storage" &&
                "is-active"
            )}
            href="/dashboard/catalog/databases-storage"
            aria-current={
              view === "catalog" && activeCategory.id === "databases-storage"
                ? "page"
                : undefined
            }
            aria-label="Data services"
            data-label="Data"
          >
            <DatabaseIcon />
          </a>
          {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
          <a
            className={cn(
              view === "catalog" &&
                activeCategory.id !== "databases-storage" &&
                "is-active"
            )}
            href="/dashboard/catalog/web-applications"
            aria-current={
              view === "catalog" && activeCategory.id !== "databases-storage"
                ? "page"
                : undefined
            }
            aria-label="Service catalog"
            data-label="Catalog"
          >
            <BoxesIcon />
          </a>
          <a href="/settings" aria-label="Settings" data-label="Settings">
            <Settings2Icon />
          </a>
          <form action="/api/auth/logout" method="post">
            <button
              className="dashboard-nav-button"
              type="submit"
              aria-label="Sign out"
              data-label="Sign out"
            >
              <LogOutIcon />
            </button>
          </form>
        </nav>
      </aside>

      <div className="dashboard-canvas">
        <header className="dashboard-header">
          <div className="admin-identity">
            <span className="admin-avatar" aria-hidden="true">
              {adminInitials || "AD"}
            </span>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="truncate text-sm font-semibold tracking-tight">
                  {admin.firstName} {admin.lastName}
                </span>
                <Badge variant="outline">Admin</Badge>
              </div>
              <p className="truncate text-xs text-muted-foreground">
                {admin.email} · global Kubernetes scope
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <div className="dashboard-wordmark" aria-label="KubeDeck">
              <KubeDeckLogo className="brand-mark" />
              <span>
                Kube<span>Deck</span>
              </span>
            </div>
            <NotificationsMenu />
            <Dialog>
              <DialogTrigger render={<Button variant="outline" size="sm" />}>
                <span className="connection-dot" aria-hidden="true" />
                Global discovery connected
              </DialogTrigger>
              <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                  <DialogTitle>Global cluster discovery</DialogTitle>
                  <DialogDescription>
                    Read-only discovery across every registered Kubernetes
                    cluster and node.
                  </DialogDescription>
                </DialogHeader>
                <div className="flex flex-col gap-4">
                  <div className="detail-grid">
                    <span>Context</span>
                    <strong>All registered clusters</strong>
                    <span>Distribution</span>
                    <strong>Kubernetes compatible</strong>
                    <span>Runtime</span>
                    <strong>Runtime independent</strong>
                    <span>Discovery</span>
                    <strong>Ingress + Service + EndpointSlice</strong>
                    <span>DNS policy</span>
                    <strong>ClusterFirst</strong>
                    <span>Captured</span>
                    <strong>{capturedAt}</strong>
                  </div>
                  <Separator />
                  <p className="text-sm leading-6 text-muted-foreground">
                    Status and run age reflect the selected global scope.
                    Continuous uptime uses cluster discovery and monitoring
                    history.
                  </p>
                </div>
                <DialogFooter showCloseButton />
              </DialogContent>
            </Dialog>
          </div>
        </header>

        <div id="dashboard-page-content" className="dashboard-page-content">
          {view !== "overview" ? (
            <section
              className="dashboard-page-heading"
              aria-labelledby="dashboard-view-title"
            >
              <div>
                <Badge variant="secondary">
                  {dashboardViewMeta[view].eyebrow}
                </Badge>
                <h1 id="dashboard-view-title">
                  {dashboardViewMeta[view].title}
                </h1>
                <p>{dashboardViewMeta[view].description}</p>
              </div>
              {view === "catalog" ? (
                <Badge variant="outline">
                  {activeCategory.label} ·{" "}
                  {
                    catalogItems.filter(
                      (item) => item.category === activeCategory.id
                    ).length
                  }{" "}
                  resources
                </Badge>
              ) : (
                <Badge variant="outline">{monitoringSnapshot.mode}</Badge>
              )}
            </section>
          ) : null}

          {view === "overview" ? (
            <>
              <section
                id="overview"
                className="dashboard-hero"
                aria-label="KubeDeck overview"
              >
                <KubeDeckBanner
                  className="dashboard-banner"
                  headingId="overview-banner-title"
                  liveGraph
                  priority
                />

                <div
                  id="fleet-resources"
                  className="resource-meter"
                  aria-labelledby="resources-title"
                >
                  <div className="flex items-start justify-between gap-6">
                    <div>
                      <p
                        id="resources-title"
                        className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground"
                      >
                        Fleet resources
                      </p>
                      <p className="mt-1 text-sm font-medium">
                        {nodeSummary.total} nodes · {nodeSummary.attention}{" "}
                        attention
                      </p>
                    </div>
                    <Badge variant="outline">
                      <CircleGaugeIcon data-icon="inline-start" />
                      Preview
                    </Badge>
                  </div>
                  <div className="resource-meter-grid">
                    {nodeResources.map((resource) => (
                      <ResourceMetric key={resource.label} {...resource} />
                    ))}
                  </div>
                  <p className="mt-3 text-[10px] leading-4 text-muted-foreground">
                    {monitoringSnapshot.source} · illustrative values at capture
                    time
                  </p>
                </div>
              </section>

              <section
                className="stat-strip"
                aria-label="Global cluster overview"
              >
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

              <nav
                className="dashboard-workspace-grid"
                aria-label="Dashboard workspaces"
              >
                <a href="/dashboard/nodes">
                  <ServerCogIcon />
                  <span>
                    <strong>Node health</strong>
                    <small>Pressure, heartbeat, and resource history</small>
                  </span>
                  <ArrowUpRightIcon />
                </a>
                <a href="/dashboard/dns">
                  <NetworkIcon />
                  <span>
                    <strong>Cluster DNS</strong>
                    <small>CoreDNS, service IP, and search domains</small>
                  </span>
                  <ArrowUpRightIcon />
                </a>
                {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
                <a href="/dashboard/catalog/web-applications">
                  <BoxesIcon />
                  <span>
                    <strong>Service catalog</strong>
                    <small>Apps, services, endpoints, and ports</small>
                  </span>
                  <ArrowUpRightIcon />
                </a>
              </nav>
            </>
          ) : null}

          {view === "nodes" ? (
            <MultiNodeReview
              showControlPlaneNodes={showControlPlaneNodes}
              showWorkerNodes={showWorkerNodes}
              highlightNodePressure={highlightNodePressure}
            />
          ) : null}

          {view === "dns" ? (
            <section className="dns-panel" aria-labelledby="dns-title">
          <div className="dns-intro">
            <span className="resource-icon" aria-hidden="true">
              <ServerCogIcon />
            </span>
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <h2 id="dns-title" className="text-lg font-semibold">
                  Cluster DNS profile
                </h2>
                <Badge variant="secondary">CoreDNS ready</Badge>
              </div>
              <p className="mt-1 text-sm leading-6 text-muted-foreground">
                Resolver and search domains for the active cluster scope.
              </p>
            </div>
          </div>
          <div className="dns-facts">
            <div>
              <span>DNS service</span>
              <strong className="font-mono">
                kube-dns.kube-system.svc.cluster.local
              </strong>
              <small>CoreDNS · 1/1 available</small>
            </div>
            <div>
              <span>DNS Service IP</span>
              <strong className="font-mono">10.43.0.10</strong>
              <small>53 UDP/TCP · metrics 9153</small>
            </div>
            <div>
              <span>Base domain</span>
              <strong className="font-mono">cluster.local</strong>
              <small>Cache 30s · ndots:5</small>
            </div>
            <div className="dns-search-path">
              <span>Pod search path</span>
              <strong className="font-mono">
                &lt;namespace&gt;.svc.cluster.local → svc.cluster.local →
                cluster.local
              </strong>
              <small>Verified from a monitoring pod</small>
            </div>
          </div>
            </section>
          ) : null}

          {view === "catalog" ? (
            <>
              <nav className="category-index" aria-label="Service categories">
          {categoryConfig.map((category) => {
            const Icon = category.icon
            const count = catalogItems.filter(
              (item) => item.category === category.id
            ).length
            return (
              <a
                key={category.id}
                className={cn(
                  activeCategory.id === category.id && "is-active"
                )}
                href={`/dashboard/catalog/${category.id}`}
                aria-current={
                  activeCategory.id === category.id ? "page" : undefined
                }
              >
                <Icon aria-hidden="true" />
                <span>{category.label}</span>
                <small>{count}</small>
              </a>
            )
          })}
              </nav>

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
              placeholder="Search name, category, DNS, IP, namespace, or port…"
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

          <div className="toolbar-filters">
            <RadioGroup
              value={kindFilter}
              onValueChange={(value) => setKindFilter(value as KindFilter)}
              aria-label="Filter by resource kind"
            >
              <Radio value="all">
                <RadioIndicator />
                All
              </Radio>
              <Radio value="app">
                <RadioIndicator />
                Web apps
              </Radio>
              <Radio value="service">
                <RadioIndicator />
                Services
              </Radio>
            </RadioGroup>

            <RadioGroup
              value={statusFilter}
              onValueChange={(value) => setStatusFilter(value as StatusFilter)}
              aria-label="Filter catalog by status"
            >
              <Radio value="all">
                <RadioIndicator />
                Any status
              </Radio>
              <Radio value="ready">
                <RadioIndicator />
                Ready
              </Radio>
              <Radio value="attention">
                <RadioIndicator />
                Attention
              </Radio>
            </RadioGroup>

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
                  <DropdownMenuLabel>Sort inside categories</DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => setSortMode("recommended")}
                  >
                    <SlidersHorizontalIcon />
                    Recommended
                    {sortMode === "recommended" && (
                      <CheckIcon className="ml-auto" />
                    )}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setSortMode("name")}>
                    <ArrowDownAZIcon />
                    Name
                    {sortMode === "name" && <CheckIcon className="ml-auto" />}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setSortMode("namespace")}>
                    <BoxesIcon />
                    Namespace
                    {sortMode === "namespace" && (
                      <CheckIcon className="ml-auto" />
                    )}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </section>

        <motion.div
          id="catalog"
          layout
          className="flex flex-col gap-14 pt-10"
        >
          <AnimatePresence mode="popLayout" initial={false}>
            {filteredItems.length === 0 ? (
              <motion.div
                key="empty-catalog"
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
              >
                <Empty>
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <SearchIcon />
                    </EmptyMedia>
                    <EmptyTitle>No cluster resources found</EmptyTitle>
                    <EmptyDescription>
                      Try another category, DNS name, IP, namespace, port, or
                      status.
                    </EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent>
                    <Button variant="outline" onClick={resetFilters}>
                      Reset filters
                    </Button>
                  </EmptyContent>
                </Empty>
              </motion.div>
            ) : (
              groupedCategories.map((category) => {
                const CategoryIcon = category.icon
                const appCount = category.items.filter(
                  (item) => item.kind === "app"
                ).length
                const serviceCount = category.items.length - appCount

                return (
                  <motion.section
                    layout
                    key={category.id}
                    id={`category-${category.id}`}
                    aria-labelledby={`category-${category.id}-title`}
                    className="scroll-mt-28"
                    initial={{ opacity: 0, y: 14 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -10 }}
                  >
                    <div className="section-heading">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="category-icon" aria-hidden="true">
                            <CategoryIcon />
                          </span>
                          <h2
                            id={`category-${category.id}-title`}
                            className="text-lg font-semibold tracking-tight"
                          >
                            {category.label}
                          </h2>
                          <Badge variant="secondary">
                            {category.items.length}
                          </Badge>
                        </div>
                        <p className="mt-2 text-sm text-muted-foreground">
                          {category.description}
                        </p>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {appCount} web {appCount === 1 ? "app" : "apps"} ·{" "}
                        {serviceCount}{" "}
                        {serviceCount === 1 ? "service" : "services"}
                      </p>
                    </div>

                    <motion.div layout className="catalog-grid">
                      <AnimatePresence mode="popLayout">
                        {category.items.map((item, index) => (
                          <ResourceCard
                            key={item.id}
                            item={item}
                            onDetails={setSelectedItem}
                            onCopy={copyClusterDns}
                          copied={copiedId === item.id}
                          index={index}
                          openInNewTab={openInNewTab}
                        />
                        ))}
                      </AnimatePresence>
                    </motion.div>
                  </motion.section>
                )
              })
            )}
          </AnimatePresence>
        </motion.div>
            </>
          ) : null}
        </div>

        <footer className="mt-16 flex flex-col gap-3 border-t border-border py-7 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
          <p>KubeDeck · Global multi-cluster Kubernetes launchpad</p>
          <p>Ingress + Service + EndpointSlice · {capturedAt}</p>
        </footer>
      </div>

      <Dialog
        open={Boolean(selectedItem)}
        onOpenChange={(open) => {
          if (!open) setSelectedItem(null)
        }}
      >
        {selectedItem && (
          <DialogContent className="sm:max-w-xl">
            <DialogHeader>
              <div className="mb-1 flex items-center gap-3">
                <span className="resource-icon" aria-hidden="true">
                  <selectedItem.icon />
                </span>
                <StatusBadge status={selectedItem.status} />
                <Badge variant="outline">
                  {getCategory(selectedItem.category).label}
                </Badge>
              </div>
              <DialogTitle>{selectedItem.name}</DialogTitle>
              <DialogDescription>{selectedItem.summary}</DialogDescription>
            </DialogHeader>

            <AvailabilityGraphic item={selectedItem} />
            <Separator />

            <div className="detail-grid">
              <span>Surface</span>
              <strong>
                {selectedItem.kind === "app" ? "Web app" : "Internal service"}
              </strong>
              <span>Namespace</span>
              <strong>{selectedItem.namespace}</strong>
              {selectedItem.externalDomain && (
                <>
                  <span>Ingress domain</span>
                  <strong className="truncate font-mono">
                    {selectedItem.externalDomain}
                  </strong>
                </>
              )}
              <span>Cluster DNS</span>
              <strong className="truncate font-mono">
                {selectedItem.internalDns}
              </strong>
              <span>Service name</span>
              <strong className="font-mono">{selectedItem.serviceName}</strong>
              <span>Service ClusterIP</span>
              <strong className="font-mono">{selectedItem.clusterIP}</strong>
              <span>Cluster domain</span>
              <strong className="font-mono">{clusterDomain}</strong>
              <span>Protocol</span>
              <strong>{selectedItem.protocol}</strong>
              <span>Ports</span>
              <strong className="font-mono">
                {selectedItem.ports.join(", ")}
              </strong>
              <span>Ready endpoints</span>
              <strong>
                {selectedItem.readyEndpoints} / {selectedItem.totalEndpoints}
              </strong>
              <span>Workload</span>
              <strong>{selectedItem.workload}</strong>
              <span>Pods</span>
              <strong>{selectedItem.pods}</strong>
              <span>Last deployed</span>
              <time
                dateTime={selectedItem.lastDeployedAt}
                className="font-medium tabular-nums"
                title={selectedItem.lastDeployedAt}
              >
                {formatUtcTimestamp(selectedItem.lastDeployedAt)}
              </time>
              <span>Discovered from</span>
              <strong>{selectedItem.source}</strong>
              <span>Captured</span>
              <strong>{capturedAt}</strong>
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => copyClusterDns(selectedItem)}
              >
                {copiedId === selectedItem.id ? (
                  <CheckIcon data-icon="inline-start" />
                ) : (
                  <CopyIcon data-icon="inline-start" />
                )}
                {copiedId === selectedItem.id
                  ? "Copied"
                  : "Copy cluster DNS"}
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
