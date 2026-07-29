"use client"

import * as React from "react"
import {
  ActivityIcon,
  BellIcon,
  CheckCheckIcon,
  CircleAlertIcon,
  DatabaseZapIcon,
  RadioTowerIcon,
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
import { cn } from "@/lib/utils"

const notifications = [
  {
    id: "discovery",
    title: "Discovery snapshot captured",
    description: "32 catalog entries across the global Kubernetes scope.",
    time: "Jul 29 · 05:21 UTC",
    icon: RadioTowerIcon,
    tone: "info",
  },
  {
    id: "workload",
    title: "Workload needs attention",
    description: "k6 Dashboard currently reports zero ready endpoints.",
    time: "Captured state",
    icon: CircleAlertIcon,
    tone: "warning",
  },
  {
    id: "dns",
    title: "Cluster DNS available",
    description: "CoreDNS is serving cluster.local through 10.43.0.10.",
    time: "System service",
    icon: DatabaseZapIcon,
    tone: "success",
  },
  {
    id: "telemetry",
    title: "Telemetry pipeline healthy",
    description: "Prometheus, Loki, Tempo, and Alloy have ready endpoints.",
    time: "Captured state",
    icon: ActivityIcon,
    tone: "success",
  },
] as const

export function NotificationsMenu() {
  const [unreadIds, setUnreadIds] = React.useState<Set<string>>(
    () => new Set(["discovery", "workload"])
  )

  function markRead(id: string) {
    setUnreadIds((current) => {
      const next = new Set(current)
      next.delete(id)
      return next
    })
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="outline"
            size="icon"
            aria-label={`${unreadIds.size} unread notifications`}
            className="notification-trigger"
          />
        }
      >
        <BellIcon />
        {unreadIds.size > 0 && (
          <span className="notification-count" aria-hidden="true">
            {unreadIds.size}
          </span>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={8}
        className="notification-menu"
      >
        <div className="flex items-center justify-between gap-4 px-2 py-2">
          <div>
            <p className="text-sm font-semibold">Notifications</p>
            <p className="text-xs text-muted-foreground">
              Kubernetes discovery events
            </p>
          </div>
          <Button
            variant="ghost"
            size="xs"
            disabled={unreadIds.size === 0}
            onClick={() => setUnreadIds(new Set())}
          >
            <CheckCheckIcon data-icon="inline-start" />
            Mark read
          </Button>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuLabel className="sr-only">
            Recent discovery notifications
          </DropdownMenuLabel>
          {notifications.map((notification) => {
            const Icon = notification.icon
            const isUnread = unreadIds.has(notification.id)

            return (
              <DropdownMenuItem
                key={notification.id}
                onClick={() => markRead(notification.id)}
                className="notification-item"
              >
                <span
                  className={cn(
                    "notification-icon",
                    `notification-icon--${notification.tone}`
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
                      <span className="notification-unread" aria-hidden="true" />
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
        <DropdownMenuSeparator />
        <p className="px-2 py-1.5 text-[10px] leading-4 text-muted-foreground">
          Captured-state notifications are labeled and are not a live alert
          stream.
        </p>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
