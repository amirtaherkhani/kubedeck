"use client"

import * as React from "react"
import { motion } from "motion/react"
import {
  ActivityIcon,
  AppWindowIcon,
  ArrowLeftIcon,
  BellRingIcon,
  BotIcon,
  CheckCircle2Icon,
  CircleGaugeIcon,
  DatabaseIcon,
  EyeIcon,
  Globe2Icon,
  KeyRoundIcon,
  LockKeyholeIcon,
  NetworkIcon,
  PencilLineIcon,
  RadioTowerIcon,
  RocketIcon,
  ServerIcon,
  ServerCogIcon,
  Settings2Icon,
  ShieldCheckIcon,
  UsersIcon,
  WorkflowIcon,
} from "lucide-react"

import { KubeDeckLogo } from "@/components/kubedeck-logo"
import { NotificationsMenu } from "@/components/notifications-menu"
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert"
import { Avatar, AvatarBadge, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldTitle,
} from "@/components/ui/field"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs"

type SettingsAdmin = {
  firstName: string
  lastName: string
  email: string
  role: "admin"
}

type Preferences = {
  motion: boolean
  compactCatalog: boolean
  externalLinks: boolean
  showControlPlaneNodes: boolean
  showWorkerNodes: boolean
  highlightNodePressure: boolean
  serviceNotifications: boolean
  nodeNotifications: boolean
  warningNotificationsOnly: boolean
}

const preferenceKeys: Record<keyof Preferences, string> = {
  motion: "kubedeck-motion-enabled",
  compactCatalog: "kubedeck-compact-catalog",
  externalLinks: "kubedeck-external-links",
  showControlPlaneNodes: "kubedeck-show-control-plane",
  showWorkerNodes: "kubedeck-show-workers",
  highlightNodePressure: "kubedeck-highlight-node-pressure",
  serviceNotifications: "kubedeck-notify-services",
  nodeNotifications: "kubedeck-notify-nodes",
  warningNotificationsOnly: "kubedeck-notify-warnings-only",
}

const categories = [
  { label: "Web Applications", count: 2, icon: AppWindowIcon },
  { label: "Databases & Storage", count: 8, icon: DatabaseIcon },
  { label: "Observability & Metrics", count: 7, icon: ActivityIcon },
  { label: "Automation & Workflows", count: 3, icon: WorkflowIcon },
  { label: "Deployments & Testing", count: 1, icon: RocketIcon },
  { label: "AI & MCP Services", count: 2, icon: BotIcon },
  { label: "Messaging & Events", count: 7, icon: RadioTowerIcon },
  { label: "Developer Tools", count: 1, icon: Settings2Icon },
  { label: "Platform & Security", count: 1, icon: ShieldCheckIcon },
] as const

const roleDefinitions = [
  {
    role: "Admin",
    description: "Full configuration, access, users, and cluster scope.",
    icon: ShieldCheckIcon,
    capabilities: "Manage all settings",
  },
  {
    role: "Editor",
    description: "Can maintain catalog metadata and application settings.",
    icon: PencilLineIcon,
    capabilities: "Edit catalog, no users",
  },
  {
    role: "Viewer",
    description: "Read-only access to services, DNS, status, and uptime.",
    icon: EyeIcon,
    capabilities: "View and launch services",
  },
] as const

export default function SettingsClient({ admin }: { admin: SettingsAdmin }) {
  const [preferences, setPreferences] = React.useState<Preferences>({
    motion: true,
    compactCatalog: false,
    externalLinks: true,
    showControlPlaneNodes: true,
    showWorkerNodes: true,
    highlightNodePressure: true,
    serviceNotifications: true,
    nodeNotifications: true,
    warningNotificationsOnly: false,
  })

  React.useEffect(() => {
    const next = {
      motion:
        window.localStorage.getItem(preferenceKeys.motion) !== "false",
      compactCatalog:
        window.localStorage.getItem(preferenceKeys.compactCatalog) === "true",
      externalLinks:
        window.localStorage.getItem(preferenceKeys.externalLinks) !== "false",
      showControlPlaneNodes:
        window.localStorage.getItem(preferenceKeys.showControlPlaneNodes) !==
        "false",
      showWorkerNodes:
        window.localStorage.getItem(preferenceKeys.showWorkerNodes) !== "false",
      highlightNodePressure:
        window.localStorage.getItem(preferenceKeys.highlightNodePressure) !==
        "false",
      serviceNotifications:
        window.localStorage.getItem(preferenceKeys.serviceNotifications) !==
        "false",
      nodeNotifications:
        window.localStorage.getItem(preferenceKeys.nodeNotifications) !==
        "false",
      warningNotificationsOnly:
        window.localStorage.getItem(
          preferenceKeys.warningNotificationsOnly
        ) === "true",
    }
    document.documentElement.classList.toggle(
      "kubedeck-motion-off",
      !next.motion
    )
    const frame = window.requestAnimationFrame(() => setPreferences(next))
    return () => window.cancelAnimationFrame(frame)
  }, [])

  function updatePreference(key: keyof Preferences, value: boolean) {
    const next = { ...preferences, [key]: value }
    setPreferences(next)
    window.localStorage.setItem(preferenceKeys[key], String(value))
    if (key === "motion") {
      document.documentElement.classList.toggle(
        "kubedeck-motion-off",
        !value
      )
    }
    window.dispatchEvent(
      new CustomEvent("kubedeck:preferences", { detail: next })
    )
  }

  const initials =
    `${admin.firstName.at(0) ?? ""}${admin.lastName.at(0) ?? ""}`.toUpperCase()

  return (
    <main id="main-content" className="settings-shell liquid-stage">
      <div className="liquid-grid" aria-hidden="true" />
      <div
        className="liquid-orbit liquid-orbit--settings"
        aria-hidden="true"
      />

      <header className="settings-header">
        <a className="settings-brand" href="/dashboard" aria-label="KubeDeck dashboard">
          <KubeDeckLogo className="brand-mark" priority />
          <span className="settings-wordmark">
            Kube<span>Deck</span>
          </span>
          <Badge variant="outline">Settings</Badge>
        </a>
        <div className="flex items-center gap-2">
          <NotificationsMenu />
          <Button
            variant="outline"
            render={<a href="/dashboard" />}
            nativeButton={false}
          >
            <ArrowLeftIcon data-icon="inline-start" />
            Dashboard
          </Button>
        </div>
      </header>

      <section className="settings-hero" aria-labelledby="settings-title">
        <div>
          <Badge variant="secondary">
            <ShieldCheckIcon data-icon="inline-start" />
            Administrator workspace
          </Badge>
          <h1 id="settings-title">Settings and access</h1>
          <p>
            Control the Kubernetes discovery scope, your KubeDeck experience,
            and the permission model from one protected workspace.
          </p>
        </div>
        <div className="settings-hero-status">
          <span className="connection-dot" aria-hidden="true" />
          <span>
            <strong>Global discovery connected</strong>
            <small>Read-only · captured Jul 29, 2026 at 05:21 UTC</small>
          </span>
        </div>
      </section>

      <Tabs defaultValue="account" className="settings-tabs">
        <TabsList className="settings-tabs-list" aria-label="Settings sections">
          <TabsTrigger value="account">
            <CircleGaugeIcon data-icon="inline-start" />
            Current user
          </TabsTrigger>
          <TabsTrigger value="kubernetes">
            <ServerCogIcon data-icon="inline-start" />
            Kubernetes
          </TabsTrigger>
          <TabsTrigger value="application">
            <Settings2Icon data-icon="inline-start" />
            App
          </TabsTrigger>
          <TabsTrigger value="users">
            <UsersIcon data-icon="inline-start" />
            Users
          </TabsTrigger>
        </TabsList>

        <TabsContent value="account">
          <motion.div
            className="settings-grid settings-grid--account"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
          >
            <Card className="settings-card settings-profile-card">
              <CardHeader>
                <CardTitle>Current user</CardTitle>
                <CardDescription>
                  Authenticated account and effective permissions.
                </CardDescription>
                <CardAction>
                  <Badge variant="secondary">Active session</Badge>
                </CardAction>
              </CardHeader>
              <CardContent className="flex flex-col gap-5">
                <div className="settings-profile">
                  <Avatar size="lg" className="size-14">
                    <AvatarFallback className="bg-primary/12 font-semibold text-primary">
                      {initials || "AD"}
                    </AvatarFallback>
                    <AvatarBadge>
                      <CheckCircle2Icon />
                    </AvatarBadge>
                  </Avatar>
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <strong className="text-base">
                        {admin.firstName} {admin.lastName}
                      </strong>
                      <Badge variant="outline">Admin</Badge>
                    </div>
                    <p className="mt-1 truncate text-sm text-muted-foreground">
                      {admin.email}
                    </p>
                  </div>
                </div>
                <Separator />
                <div className="settings-facts">
                  <span>Effective role</span>
                  <strong>Administrator</strong>
                  <span>Cluster scope</span>
                  <strong>All registered clusters</strong>
                  <span>Session policy</span>
                  <strong>HTTP-only · 12 hours</strong>
                  <span>Authentication</span>
                  <strong>PBKDF2 secured account</strong>
                </div>
              </CardContent>
            </Card>

            <Card className="settings-card">
              <CardHeader>
                <CardTitle>Permission boundary</CardTitle>
                <CardDescription>
                  What the current administrator can control.
                </CardDescription>
              </CardHeader>
              <CardContent className="settings-capability-list">
                {[
                  ["Open discovered apps and services", Globe2Icon],
                  ["View DNS, ports, status, and uptime", NetworkIcon],
                  ["Configure app preferences", Settings2Icon],
                  ["Manage access policy and users", UsersIcon],
                ].map(([label, Icon]) => (
                  <div key={String(label)}>
                    <span className="settings-capability-icon">
                      <Icon />
                    </span>
                    <span>{String(label)}</span>
                    <CheckCircle2Icon className="ml-auto text-emerald-400" />
                  </div>
                ))}
              </CardContent>
            </Card>
          </motion.div>
        </TabsContent>

        <TabsContent value="kubernetes">
          <div className="settings-grid">
            <Card className="settings-card">
              <CardHeader>
                <CardTitle>Kubernetes discovery</CardTitle>
                <CardDescription>
                  Active read-only inventory configuration.
                </CardDescription>
                <CardAction>
                  <Badge variant="secondary">
                    <span className="connection-dot" aria-hidden="true" />
                    Connected
                  </Badge>
                </CardAction>
              </CardHeader>
              <CardContent className="settings-facts">
                <span>Context</span>
                <strong>All registered clusters</strong>
                <span>Resources</span>
                <strong>Ingress, Service, EndpointSlice</strong>
                <span>Workloads</span>
                <strong>Deployment, StatefulSet, DaemonSet, Pod</strong>
                <span>Access mode</span>
                <strong>Read-only discovery</strong>
                <span>Refresh source</span>
                <strong>Captured inventory</strong>
              </CardContent>
            </Card>

            <Card className="settings-card">
              <CardHeader>
                <CardTitle>Cluster DNS</CardTitle>
                <CardDescription>
                  DNS policy and system resolver identity.
                </CardDescription>
              </CardHeader>
              <CardContent className="settings-facts">
                <span>Cluster domain</span>
                <strong className="font-mono">cluster.local</strong>
                <span>Base DNS service</span>
                <strong className="font-mono">
                  kube-dns.kube-system.svc.cluster.local
                </strong>
                <span>DNS service IP</span>
                <strong className="font-mono">10.43.0.10</strong>
                <span>Pod DNS policy</span>
                <strong>ClusterFirst</strong>
              </CardContent>
            </Card>

            <Card className="settings-card settings-card--wide">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <ServerIcon />
                  Multi-node cluster review
                </CardTitle>
                <CardDescription>
                  Choose which Kubernetes node roles appear in the home review
                  and how pressure conditions are emphasized.
                </CardDescription>
                <CardAction>
                  <Badge variant="outline">3 sample nodes · 4 signals</Badge>
                </CardAction>
              </CardHeader>
              <CardContent>
                <FieldGroup className="settings-preferences">
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Show control-plane nodes</FieldTitle>
                      <FieldDescription>
                        Include scheduler and control-plane hosts in the node
                        table and chart selector.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.showControlPlaneNodes}
                      onCheckedChange={(checked) =>
                        updatePreference("showControlPlaneNodes", checked)
                      }
                      aria-label="Show control-plane nodes"
                    />
                  </Field>
                  <Separator />
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Show worker nodes</FieldTitle>
                      <FieldDescription>
                        Include application and data workers in the node review.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.showWorkerNodes}
                      onCheckedChange={(checked) =>
                        updatePreference("showWorkerNodes", checked)
                      }
                      aria-label="Show worker nodes"
                    />
                  </Field>
                  <Separator />
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Highlight node pressure</FieldTitle>
                      <FieldDescription>
                        Emphasize memory, disk, PID, or readiness conditions in
                        the node status table.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.highlightNodePressure}
                      onCheckedChange={(checked) =>
                        updatePreference("highlightNodePressure", checked)
                      }
                      aria-label="Highlight node pressure conditions"
                    />
                  </Field>
                </FieldGroup>
                <Alert className="mt-5 border-primary/25 bg-primary/5">
                  <ActivityIcon />
                  <AlertTitle>Telemetry source</AlertTitle>
                  <AlertDescription>
                    The current screen uses an illustrative multi-node snapshot.
                    Connect metrics-server plus kubelet summary data, or
                    Prometheus history, before treating the four-property chart
                    as live telemetry.
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>

            <Card className="settings-card settings-card--wide">
              <CardHeader>
                <CardTitle>Intelligent service taxonomy</CardTitle>
                <CardDescription>
                  Every catalog entry is assigned by its operational purpose.
                </CardDescription>
                <CardAction>
                  <Badge variant="outline">32 services</Badge>
                </CardAction>
              </CardHeader>
              <CardContent className="settings-category-grid">
                {categories.map((category) => {
                  const Icon = category.icon
                  return (
                    <div key={category.label}>
                      <span className="settings-category-icon">
                        <Icon />
                      </span>
                      <span className="min-w-0">
                        <strong>{category.label}</strong>
                        <small>{category.count} catalog entries</small>
                      </span>
                    </div>
                  )
                })}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="application">
          <div className="settings-grid">
            <Card className="settings-card settings-card--wide">
              <CardHeader>
                <CardTitle>Application preferences</CardTitle>
                <CardDescription>
                  Saved on this device and applied across KubeDeck.
                </CardDescription>
                <CardAction>
                  <Badge variant="outline">Local preferences</Badge>
                </CardAction>
              </CardHeader>
              <CardContent>
                <FieldGroup className="settings-preferences">
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Interface motion</FieldTitle>
                      <FieldDescription>
                        Enable topology flow, surface transitions, and animated
                        status feedback.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.motion}
                      onCheckedChange={(checked) =>
                        updatePreference("motion", checked)
                      }
                      aria-label="Enable interface motion"
                    />
                  </Field>
                  <Separator />
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Compact service catalog</FieldTitle>
                      <FieldDescription>
                        Reduce card spacing to fit more Kubernetes services on
                        screen.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.compactCatalog}
                      onCheckedChange={(checked) =>
                        updatePreference("compactCatalog", checked)
                      }
                      aria-label="Use compact service catalog"
                    />
                  </Field>
                  <Separator />
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Open service URLs in a new tab</FieldTitle>
                      <FieldDescription>
                        Keep KubeDeck open while launching an external web
                        application.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.externalLinks}
                      onCheckedChange={(checked) =>
                        updatePreference("externalLinks", checked)
                      }
                      aria-label="Open service URLs in a new tab"
                    />
                  </Field>
                </FieldGroup>
              </CardContent>
            </Card>

            <Card className="settings-card">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <BellRingIcon />
                  Notification policy
                </CardTitle>
                <CardDescription>
                  Control which captured service and node states appear in the
                  notification menu.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <FieldGroup className="settings-preferences">
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Service status notifications</FieldTitle>
                      <FieldDescription>
                        Report endpoint readiness and telemetry service health.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.serviceNotifications}
                      onCheckedChange={(checked) =>
                        updatePreference("serviceNotifications", checked)
                      }
                      aria-label="Enable service status notifications"
                    />
                  </Field>
                  <Separator />
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Node status notifications</FieldTitle>
                      <FieldDescription>
                        Report readiness, heartbeat, and pressure conditions.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.nodeNotifications}
                      onCheckedChange={(checked) =>
                        updatePreference("nodeNotifications", checked)
                      }
                      aria-label="Enable node status notifications"
                    />
                  </Field>
                  <Separator />
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>Warnings only</FieldTitle>
                      <FieldDescription>
                        Hide healthy summaries and keep only states that need
                        attention.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      checked={preferences.warningNotificationsOnly}
                      onCheckedChange={(checked) =>
                        updatePreference("warningNotificationsOnly", checked)
                      }
                      aria-label="Show warning notifications only"
                    />
                  </Field>
                </FieldGroup>
              </CardContent>
            </Card>

            <Card className="settings-card">
              <CardHeader>
                <CardTitle>Security defaults</CardTitle>
                <CardDescription>
                  Built-in protections for private operation.
                </CardDescription>
              </CardHeader>
              <CardContent className="settings-capability-list">
                {[
                  ["Private access", LockKeyholeIcon],
                  ["12-hour sessions", KeyRoundIcon],
                  ["External link isolation", Globe2Icon],
                ].map(([label, Icon]) => (
                  <div key={String(label)}>
                    <span className="settings-capability-icon">
                      <Icon />
                    </span>
                    <span>{String(label)}</span>
                    <CheckCircle2Icon className="ml-auto text-emerald-400" />
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="users">
          <div className="settings-grid">
            <Alert className="settings-card--wide border-primary/25 bg-primary/5">
              <UsersIcon />
              <AlertTitle>Single-admin authentication is active</AlertTitle>
              <AlertDescription>
                The current account is fully manageable as an administrator.
                Editor and Viewer define the production RBAC model; provisioning
                those accounts requires a multi-user identity store or SSO
                integration so KubeDeck never displays fictional users.
              </AlertDescription>
            </Alert>

            <Card className="settings-card settings-card--wide">
              <CardHeader>
                <CardTitle>Users management</CardTitle>
                <CardDescription>
                  Accounts currently authorized to open this private workspace.
                </CardDescription>
                <CardAction>
                  <Badge variant="secondary">1 active user</Badge>
                </CardAction>
              </CardHeader>
              <CardContent>
                <div className="settings-user-row">
                  <Avatar size="lg">
                    <AvatarFallback>{initials || "AD"}</AvatarFallback>
                    <AvatarBadge />
                  </Avatar>
                  <span className="min-w-0">
                    <strong>
                      {admin.firstName} {admin.lastName}
                    </strong>
                    <small>{admin.email}</small>
                  </span>
                  <Badge variant="outline">Admin</Badge>
                  <Badge variant="secondary">Active</Badge>
                </div>
              </CardContent>
            </Card>

            {roleDefinitions.map((definition) => {
              const Icon = definition.icon
              return (
                <Card key={definition.role} className="settings-card">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <span className="settings-capability-icon">
                        <Icon />
                      </span>
                      {definition.role}
                    </CardTitle>
                    <CardDescription>
                      {definition.description}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <Badge
                      variant={
                        definition.role === "Admin" ? "secondary" : "outline"
                      }
                    >
                      {definition.capabilities}
                    </Badge>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </TabsContent>
      </Tabs>
    </main>
  )
}
