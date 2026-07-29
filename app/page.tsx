import {
  BoxesIcon,
  Globe2Icon,
  LockKeyholeIcon,
  NetworkIcon,
  ServerCogIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { redirect } from "next/navigation"

import { AdminAuthForm } from "@/components/admin-auth-form"
import { KubeDeckBanner } from "@/components/kubedeck-banner"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { getConfiguredAdmin, getCurrentAdmin } from "@/lib/auth"

export const dynamic = "force-dynamic"

export default async function LoginPage() {
  const admin = await getConfiguredAdmin()
  if (admin && (await getCurrentAdmin())) redirect("/dashboard")

  const isSetup = !admin

  return (
    <main className="login-shell liquid-stage">
      <div className="liquid-grid" aria-hidden="true" />
      <div className="liquid-orbit liquid-orbit--login" aria-hidden="true" />

      <div className="login-frame">
        <section className="login-showcase" aria-labelledby="login-heading">
          <div className="login-brand-row">
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
                  Kubernetes services, one click away
                </p>
              </div>
            </div>
            <span className="login-connection">
              <span className="connection-dot" aria-hidden="true" />
              Discovery ready
            </span>
          </div>

          <KubeDeckBanner
            className="login-banner"
            headingId="login-heading"
            priority
          />

          <div className="login-scope-strip" aria-label="KubeDeck scope">
            <span>
              <Globe2Icon aria-hidden="true" />
              Multi-cluster
            </span>
            <span>
              <ServerCogIcon aria-hidden="true" />
              All nodes
            </span>
            <span>
              <NetworkIcon aria-hidden="true" />
              Global services
            </span>
          </div>
        </section>

        <Card className="login-card liquid-glass-card [--card-spacing:--spacing(6)]">
          <CardHeader>
            <CardTitle>
              {isSetup ? "Create the admin account" : "Sign in to KubeDeck"}
            </CardTitle>
            <CardDescription>
              {isSetup
                ? "Complete this one-time setup with the administrator’s name, email, and password."
                : "Use the configured administrator email and password to open the cluster dashboard."}
            </CardDescription>
            <CardAction>
              <Badge variant={isSetup ? "secondary" : "outline"}>
                {isSetup ? (
                  <ShieldCheckIcon data-icon="inline-start" />
                ) : (
                  <LockKeyholeIcon data-icon="inline-start" />
                )}
                {isSetup ? "First setup" : "Private"}
              </Badge>
            </CardAction>
          </CardHeader>

          <CardContent>
            <AdminAuthForm mode={isSetup ? "setup" : "login"} />
          </CardContent>

          <CardFooter>
            <p className="text-center text-[11px] text-muted-foreground">
              Passwords are salted and hashed. The dashboard session is
              HTTP-only and expires after 12 hours.
            </p>
          </CardFooter>
        </Card>
      </div>
    </main>
  )
}
