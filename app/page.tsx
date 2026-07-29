import {
  ArrowRightIcon,
  BoxesIcon,
  CircleCheckIcon,
  CloudCogIcon,
  Globe2Icon,
  LockKeyholeIcon,
  NetworkIcon,
  ShieldCheckIcon,
  UserRoundIcon,
} from "lucide-react"
import { headers } from "next/headers"
import Link from "next/link"

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
import { Separator } from "@/components/ui/separator"

export const dynamic = "force-dynamic"

function decodeDisplayName(
  value: string | null,
  encoding: string | null
): string | null {
  if (!value || encoding !== "percent-encoded-utf-8") return null

  try {
    return decodeURIComponent(value)
  } catch {
    return null
  }
}

export default async function LoginPage() {
  const requestHeaders = await headers()
  const email = requestHeaders.get("oai-authenticated-user-email")
  const fullName = decodeDisplayName(
    requestHeaders.get("oai-authenticated-user-full-name"),
    requestHeaders.get("oai-authenticated-user-full-name-encoding")
  )
  const displayName = fullName ?? email?.split("@")[0] ?? "Private operator"
  const identityDetail = email ?? "Owner-only Sites access"

  return (
    <main className="login-shell">
      <div className="login-frame">
        <section className="login-intro" aria-labelledby="login-heading">
          <div className="flex items-center gap-3">
            <span className="brand-mark" aria-hidden="true">
              <BoxesIcon />
            </span>
            <div>
              <p className="text-base font-semibold tracking-tight">KubeDeck</p>
              <p className="text-xs text-muted-foreground">
                Kubernetes service launchpad
              </p>
            </div>
          </div>

          <div>
            <Badge variant="secondary">
              <ShieldCheckIcon data-icon="inline-start" />
              Owner-only workspace
            </Badge>
            <h1
              id="login-heading"
              className="mt-5 max-w-xl text-4xl font-semibold tracking-[-0.045em] text-balance sm:text-5xl"
            >
              One secure entry to your
              <span className="text-primary"> cluster cockpit.</span>
            </h1>
            <p className="mt-4 max-w-xl text-base leading-7 text-muted-foreground">
              Open web apps, inspect internal DNS and ports, and scan service
              health across your Rancher Desktop Kubernetes environment.
            </p>
          </div>

          <div className="login-facts" aria-label="Workspace summary">
            <div>
              <CircleCheckIcon aria-hidden="true" />
              <span>
                <small>Node status</small>
                <strong>1 ready</strong>
              </span>
            </div>
            <div>
              <CloudCogIcon aria-hidden="true" />
              <span>
                <small>Catalog</small>
                <strong>32 endpoints</strong>
              </span>
            </div>
            <div>
              <NetworkIcon aria-hidden="true" />
              <span>
                <small>Cluster DNS</small>
                <strong>cluster.local</strong>
              </span>
            </div>
          </div>
        </section>

        <Card className="login-card [--card-spacing:--spacing(6)]">
          <CardHeader>
            <CardTitle>Sign in to KubeDeck</CardTitle>
            <CardDescription>
              Continue with the private identity already verified by this
              workspace.
            </CardDescription>
            <CardAction>
              <Badge variant="outline">
                <LockKeyholeIcon data-icon="inline-start" />
                Private
              </Badge>
            </CardAction>
          </CardHeader>

          <CardContent className="flex flex-col gap-5">
            <div className="login-identity">
              <span className="resource-icon" aria-hidden="true">
                <UserRoundIcon />
              </span>
              <span className="min-w-0 flex-1">
                <small>Verified identity</small>
                <strong>{displayName}</strong>
                <span>{identityDetail}</span>
              </span>
              <CircleCheckIcon
                className="size-5 shrink-0 text-primary"
                aria-label="Identity verified"
              />
            </div>

            <div className="login-connection">
              <div>
                <Globe2Icon aria-hidden="true" />
                <span>Workspace</span>
                <strong>Private access</strong>
              </div>
              <div>
                <CloudCogIcon aria-hidden="true" />
                <span>Context</span>
                <strong>rancher-desktop</strong>
              </div>
              <div>
                <NetworkIcon aria-hidden="true" />
                <span>Discovery</span>
                <strong>Read-only snapshot</strong>
              </div>
            </div>

            <Separator />

            <p className="text-xs leading-5 text-muted-foreground">
              KubeDeck does not request or store Rancher tokens, kubeconfig
              files, or cluster credentials on this screen.
            </p>
          </CardContent>

          <CardFooter className="flex-col gap-3">
            <Button
              size="lg"
              className="w-full"
              render={<Link href="/dashboard" />}
              nativeButton={false}
            >
              Continue to dashboard
              <ArrowRightIcon data-icon="inline-end" />
            </Button>
            <p className="text-center text-[11px] text-muted-foreground">
              Access is enforced before this page loads.
            </p>
          </CardFooter>
        </Card>
      </div>
    </main>
  )
}
