import {
  ArrowRightIcon,
  CircleCheckIcon,
  CloudCogIcon,
  Globe2Icon,
  LockKeyholeIcon,
  NetworkIcon,
  UserRoundIcon,
} from "lucide-react"
import { headers } from "next/headers"
import Image from "next/image"
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
        <section className="login-banner" aria-labelledby="login-heading">
          <h1 id="login-heading" className="sr-only">
            KubeDeck — your Kubernetes ecosystem, one click away
          </h1>
          <Image
            src="/og.png"
            width={1731}
            height={909}
            alt="KubeDeck Kubernetes ecosystem banner showing connected web apps and services"
            priority
            sizes="(max-width: 900px) calc(100vw - 32px), 62vw"
          />
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
