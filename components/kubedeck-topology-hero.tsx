import {
  BoxesIcon,
  CloudCogIcon,
  DatabaseIcon,
  Globe2Icon,
  NetworkIcon,
  ServerCogIcon,
  WaypointsIcon,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"

const topologyNodes = [
  { className: "topology-node--north", icon: CloudCogIcon },
  { className: "topology-node--northwest", icon: NetworkIcon },
  { className: "topology-node--northeast", icon: Globe2Icon },
  { className: "topology-node--southwest", icon: DatabaseIcon },
  { className: "topology-node--south", icon: ServerCogIcon },
  { className: "topology-node--southeast", icon: BoxesIcon },
] as const

const topologyRoutes = [
  "topology-route--north",
  "topology-route--northwest",
  "topology-route--northeast",
  "topology-route--southwest",
  "topology-route--south",
  "topology-route--southeast",
] as const

export function KubeDeckTopologyHero() {
  return (
    <section className="login-topology" aria-labelledby="login-heading">
      <div className="topology-copy">
        <Badge variant="secondary" className="w-fit">
          <Globe2Icon data-icon="inline-start" />
          Global multi-cluster
        </Badge>
        <h1 id="login-heading" className="topology-wordmark">
          <span>Kube</span>Deck
        </h1>
        <p className="topology-summary">
          All Kubernetes clusters, nodes, apps, and services—one click away.
        </p>
        <div className="topology-scope-list" aria-label="KubeDeck scope">
          <span>Any distribution</span>
          <span>Every node</span>
          <span>Global DNS and routes</span>
        </div>
      </div>

      <div
        className="topology-visual"
        role="img"
        aria-label="Animated global Kubernetes topology connecting multiple clusters, nodes, applications, and services to KubeDeck"
      >
        <div className="topology-grid" aria-hidden="true" />
        <div className="topology-orbit topology-orbit--outer" aria-hidden="true" />
        <div className="topology-orbit topology-orbit--inner" aria-hidden="true" />

        {topologyRoutes.map((route) => (
          <span
            key={route}
            className={`topology-route ${route}`}
            aria-hidden="true"
          >
            <i />
          </span>
        ))}

        <span className="topology-node topology-node--core" aria-hidden="true">
          <WaypointsIcon />
          <small>KubeDeck</small>
        </span>

        {topologyNodes.map(({ className, icon: Icon }) => (
          <span
            key={className}
            className={`topology-node ${className}`}
            aria-hidden="true"
          >
            <Icon />
          </span>
        ))}

        <div className="topology-legend" aria-hidden="true">
          <span>
            <i />
            Clusters
          </span>
          <span>
            <i />
            Nodes
          </span>
          <span>
            <i />
            Services
          </span>
        </div>
      </div>
    </section>
  )
}
