import { LockKeyholeIcon, ShieldCheckIcon } from "lucide-react"
import Image from "next/image"
import { redirect } from "next/navigation"

import { AdminAuthForm } from "@/components/admin-auth-form"
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
import { getAdmin, getCurrentAdmin } from "@/lib/auth"

export const dynamic = "force-dynamic"

export default async function LoginPage() {
  const admin = await getAdmin()
  if (admin && (await getCurrentAdmin())) redirect("/dashboard")

  const isSetup = !admin

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
