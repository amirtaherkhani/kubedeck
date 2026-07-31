"use client"

import * as React from "react"
import type { LucideIcon } from "lucide-react"
import {
  ActivityIcon,
  BellIcon,
  BoxesIcon,
  CheckCheckIcon,
  CircleAlertIcon,
  CircleCheckIcon,
  MemoryStickIcon,
  ServerIcon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  buildLiveNodes,
  relativeTimestamp,
  type ClusterSnapshot,
} from "@/lib/kubedeck-cluster"
import type { ClusterConnectionStatus } from "@/lib/use-kubedeck-cluster"
import { cn } from "@/lib/utils"

type NotificationScope = "service" | "node"
type NotificationSeverity = "warning" | "success"

type StatusNotification = {
  id: string
  scope: NotificationScope
  severity: NotificationSeverity
  title: string
  description: string
  time: string
  icon: LucideIcon
}

const notificationPreferenceKeys = {
  service: "kubedeck-notify-services",
  node: "kubedeck-notify-nodes",
  warningsOnly: "kubedeck-notify-warnings-only",
  read: "kubedeck-read-notifications",
} as const

function readStoredIds() {
  try {
    const value = window.localStorage.getItem(notificationPreferenceKeys.read)
    const parsed = value ? (JSON.parse(value) as unknown) : []
    return Array.isArray(parsed)
      ? new Set(parsed.filter((id): id is string => typeof id === "string"))
      : new Set<string>()
  } catch {
    return new Set<string>()
  }
}

export function NotificationsMenu({
  snapshot,
  connectionStatus,
}: {
  snapshot: ClusterSnapshot | null
  connectionStatus: ClusterConnectionStatus
}) {
  const [readIds, setReadIds] = React.useState<Set<string>>(() => new Set())
  const [preferences, setPreferences] = React.useState({
    service: true,
    node: true,
    warningsOnly: false,
  })
  const notifications = React.useMemo(
    () => buildNotifications(snapshot, connectionStatus),
    [connectionStatus, snapshot]
  )

  React.useEffect(() => {
    function readPreferences() {
      const nextPreferences = {
        service:
          window.localStorage.getItem(notificationPreferenceKeys.service) !==
          "false",
        node:
          window.localStorage.getItem(notificationPreferenceKeys.node) !==
          "false",
        warningsOnly:
          window.localStorage.getItem(
            notificationPreferenceKeys.warningsOnly
          ) === "true",
      }
      const nextReadIds = readStoredIds()
      const frame = window.requestAnimationFrame(() => {
        setPreferences(nextPreferences)
        setReadIds(nextReadIds)
      })
      return () => window.cancelAnimationFrame(frame)
    }

    const cancelFrame = readPreferences()
    window.addEventListener("kubedeck:preferences", readPreferences)
    return () => {
      cancelFrame()
      window.removeEventListener("kubedeck:preferences", readPreferences)
    }
  }, [])

  const visibleNotifications = notifications.filter(
    (notification) =>
      preferences[notification.scope] &&
      (!preferences.warningsOnly || notification.severity === "warning")
  )
  const unreadCount = visibleNotifications.filter(
    (notification) => !readIds.has(notification.id)
  ).length
  const serviceWarningCount = visibleNotifications.filter(
    (notification) =>
      notification.scope === "service" &&
      notification.severity === "warning"
  ).length
  const nodeWarningCount = visibleNotifications.filter(
    (notification) =>
      notification.scope === "node" && notification.severity === "warning"
  ).length

  function updateReadIds(next: Set<string>) {
    setReadIds(next)
    window.localStorage.setItem(
      notificationPreferenceKeys.read,
      JSON.stringify([...next])
    )
  }

  function markRead(id: string) {
    const next = new Set(readIds)
    next.add(id)
    updateReadIds(next)
  }

  function markVisibleRead() {
    const next = new Set(readIds)
    for (const notification of visibleNotifications) {
      next.add(notification.id)
    }
    updateReadIds(next)
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="outline"
            size="icon"
            aria-label={`${unreadCount} unread status notifications`}
            className="notification-trigger"
          />
        }
      >
        <BellIcon />
        {unreadCount > 0 && (
          <span className="notification-count" aria-hidden="true">
            {unreadCount}
          </span>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={8}
        className="notification-menu"
      >
        <div className="notification-menu-header">
          <div>
            <p className="text-sm font-semibold">Status notifications</p>
            <p className="text-xs text-muted-foreground">
              Services and Kubernetes nodes
            </p>
          </div>
          <Button
            variant="ghost"
            size="xs"
            disabled={unreadCount === 0}
            onClick={markVisibleRead}
          >
            <CheckCheckIcon data-icon="inline-start" />
            Mark read
          </Button>
        </div>

        <div className="notification-summary" aria-label="Attention summary">
          <span>
            <BoxesIcon aria-hidden="true" />
            {serviceWarningCount} service
          </span>
          <span>
            <ServerIcon aria-hidden="true" />
            {nodeWarningCount} node
          </span>
        </div>

        <DropdownMenuSeparator />
        {visibleNotifications.length > 0 ? (
          (["service", "node"] as const).map((scope) => {
            const scopedNotifications = visibleNotifications.filter(
              (notification) => notification.scope === scope
            )
            if (scopedNotifications.length === 0) return null

            return (
              <DropdownMenuGroup key={scope}>
                <DropdownMenuLabel>
                  {scope === "service" ? "Services" : "Nodes"}
                </DropdownMenuLabel>
                {scopedNotifications.map((notification) => {
                  const Icon = notification.icon
                  const isUnread = !readIds.has(notification.id)

                  return (
                    <DropdownMenuItem
                      key={notification.id}
                      onClick={() => markRead(notification.id)}
                      className="notification-item"
                    >
                      <span
                        className={cn(
                          "notification-icon",
                          `notification-icon--${notification.severity}`
                        )}
                        aria-hidden="true"
                      >
                        <Icon />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="flex items-center gap-2">
                          <strong className="truncate text-xs font-semibold">
                            {notification.title}
                          </strong>
                          {isUnread && (
                            <span
                              className="notification-unread"
                              aria-hidden="true"
                            />
                          )}
                        </span>
                        <span className="mt-1 block text-xs leading-4 text-muted-foreground">
                          {notification.description}
                        </span>
                        <span className="mt-1.5 block font-mono text-[10px] text-muted-foreground">
                          {notification.time}
                        </span>
                      </span>
                    </DropdownMenuItem>
                  )
                })}
              </DropdownMenuGroup>
            )
          })
        ) : (
          <Empty className="min-h-40">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <CircleCheckIcon />
              </EmptyMedia>
              <EmptyTitle>No notification sources enabled</EmptyTitle>
              <EmptyDescription>
                Enable service or node notifications in App Settings.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
        <DropdownMenuSeparator />
        <p className="px-2 py-1.5 text-[10px] leading-4 text-muted-foreground">
          {snapshot
            ? "Live kubedeck-agent snapshot and Kubernetes warning events."
            : "Waiting for a live kubedeck-agent snapshot."}
        </p>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function buildNotifications(
  snapshot: ClusterSnapshot | null,
  connectionStatus: ClusterConnectionStatus
): StatusNotification[] {
  if (!snapshot) {
    return [
      {
        id: "agent-unavailable",
        scope: "service",
        severity: "warning",
        title: "Cluster agent is not connected",
        description:
          "KubeDeck is waiting for an authenticated snapshot and live event stream.",
        time:
          connectionStatus === "unavailable"
            ? "Agent unavailable"
            : "Connecting",
        icon: CircleAlertIcon,
      },
    ]
  }

  const result: StatusNotification[] = []
  const generatedAt = Date.parse(snapshot.generatedAt)
  for (const service of snapshot.services
    .filter((item) => item.status !== "ready")
    .slice(0, 4)) {
    result.push({
      id: `service-${service.uid || `${service.namespace}-${service.name}`}`,
      scope: "service",
      severity: "warning",
      title: `${service.namespace}/${service.name} needs attention`,
      description: `${service.readyEndpoints}/${service.totalEndpoints} endpoints and ${service.readyPods}/${service.totalPods} pods are ready.`,
      time: "Latest agent snapshot",
      icon: CircleAlertIcon,
    })
  }

  for (const node of buildLiveNodes(snapshot)
    .filter((item) => item.status === "attention")
    .slice(0, 4)) {
    result.push({
      id: `node-${node.id}`,
      scope: "node",
      severity: "warning",
      title: `${node.name} needs attention`,
      description: `${node.condition}; CPU ${node.cpu}%, memory ${node.memory}%, pod allocation ${node.podAllocation}%.`,
      time: node.heartbeat,
      icon: MemoryStickIcon,
    })
  }

  for (const event of snapshot.events
    .filter((item) => item.type.toLowerCase() === "warning")
    .slice(0, 4)) {
    const nodeEvent = event.regardingKind?.toLowerCase() === "node"
    result.push({
      id: `event-${event.uid || event.name}`,
      scope: nodeEvent ? "node" : "service",
      severity: "warning",
      title: event.reason || "Kubernetes warning event",
      description: [
        event.regardingNamespace,
        event.regardingName,
        event.message,
      ]
        .filter(Boolean)
        .join(" · "),
      time: relativeTimestamp(
        event.lastSeenAt || event.firstSeenAt,
        Number.isFinite(generatedAt) ? generatedAt : Date.now()
      ),
      icon: CircleAlertIcon,
    })
  }

  result.push(
    {
      id: "service-ready-summary",
      scope: "service",
      severity: "success",
      title: "Service readiness summary",
      description: `${snapshot.summary.readyServices} of ${snapshot.summary.services} Services have ready endpoints.`,
      time: "Latest agent snapshot",
      icon: ActivityIcon,
    },
    {
      id: "node-ready-summary",
      scope: "node",
      severity: "success",
      title: "Node readiness summary",
      description: `${snapshot.summary.readyNodes} of ${snapshot.summary.nodes} nodes report Kubernetes Ready.`,
      time: "Latest agent snapshot",
      icon: ServerIcon,
    }
  )
  return result
}
