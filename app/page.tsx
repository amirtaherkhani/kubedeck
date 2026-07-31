import {
  BoxesIcon,
  CircleCheckIcon,
  ContainerIcon,
  DatabaseIcon,
  Globe2Icon,
  LockKeyholeIcon,
  NetworkIcon,
  ShieldCheckIcon,
  SparklesIcon,
} from "lucide-react"
import { redirect } from "next/navigation"

import { AdminAuthForm } from "@/components/admin-auth-form"
import { KubeDeckLogo } from "@/components/kubedeck-logo"
import { Badge } from "@/components/ui/badge"
import { getConfiguredAdmin, getCurrentAdmin } from "@/lib/auth"

export const dynamic = "force-dynamic"

export default async function LoginPage() {
  const admin = await getConfiguredAdmin()
  if (admin && (await getCurrentAdmin())) redirect("/dashboard")

  const isSetup = !admin

  return (
    <main
      className={`auth-shell liquid-stage ${
        isSetup ? "auth-shell--setup" : "auth-shell--login"
      }`}
    >
      <div className="auth-starfield" aria-hidden="true" />
      <div className="auth-layout">
        <section className="auth-form-panel" aria-labelledby="auth-heading">
          <header className="auth-brand-row">
            <div className="auth-brand-lockup">
              <KubeDeckLogo className="auth-brand-mark" priority />
              <div>
                <span>KubeDeck</span>
                <small>Kubernetes launchpad</small>
              </div>
            </div>
            <Badge variant="outline" className="auth-private-badge">
              <span className="connection-dot" aria-hidden="true" />
              Private
            </Badge>
          </header>

          <div className="auth-form-wrap">
            <div className="auth-heading-group">
              <div className="auth-kicker">
                {isSetup ? (
                  <SparklesIcon aria-hidden="true" />
                ) : (
                  <LockKeyholeIcon aria-hidden="true" />
                )}
                {isSetup ? "One-time setup" : "Admin access"}
              </div>
              <h1 id="auth-heading">
                {isSetup ? "Create the admin account" : "Sign in to KubeDeck"}
              </h1>
              <p>
                {isSetup
                  ? "Make yourself at home. Add the first administrator and your private cluster deck is ready to explore."
                  : "Welcome back, cluster keeper. Your services, nodes, and routes are right where you left them."}
              </p>
            </div>

            <AdminAuthForm mode={isSetup ? "setup" : "login"} />

            <div className="auth-security-note">
              <ShieldCheckIcon aria-hidden="true" />
              <p>
                <strong>Private by design.</strong>
                KubeDeck stores only a salted hash. Sessions are HTTP-only and
                expire after 12 hours.
              </p>
            </div>
          </div>

          <footer className="auth-form-footer">
            <span>© 2026 KubeDeck</span>
            <span>Single-admin workspace</span>
          </footer>
        </section>

        <section className="auth-story-panel" aria-labelledby="story-heading">
          <div className="auth-silk auth-silk--one" aria-hidden="true" />
          <div className="auth-silk auth-silk--two" aria-hidden="true" />

          <div className="auth-story-topbar">
            <span>
              <CircleCheckIcon aria-hidden="true" />
              Discovery ready
            </span>
            <span>01 / ADMIN</span>
          </div>

          <div className="auth-story-copy">
            <span className="auth-story-eyebrow">
              {isSetup ? "A fresh deck awaits" : "Everything is still humming"}
            </span>
            <h2 id="story-heading">
              All your clusters.
              <em>One happy little deck.</em>
            </h2>
            <p>
              A calm home for the noisy parts of Kubernetes—gathered, sorted,
              and always one click away.
            </p>
          </div>

          <div className="auth-cluster-scene" aria-hidden="true">
            <span className="auth-route auth-route--one"><i /></span>
            <span className="auth-route auth-route--two"><i /></span>
            <span className="auth-route auth-route--three"><i /></span>
            <span className="auth-route auth-route--four"><i /></span>

            <div className="auth-cluster-orbit auth-cluster-orbit--outer" />
            <div className="auth-cluster-orbit auth-cluster-orbit--inner" />

            <div className="auth-cluster-core">
              <span className="auth-core-glow" />
              <KubeDeckLogo className="auth-core-mark" priority />
              <span className="auth-core-smile">⌣</span>
              <small>Your deck</small>
            </div>

            <div className="auth-service-pill auth-service-pill--nodes">
              <span><ContainerIcon /></span>
              <div><strong>Nodes</strong><small>Healthy & cozy</small></div>
            </div>
            <div className="auth-service-pill auth-service-pill--dns">
              <span><NetworkIcon /></span>
              <div><strong>Cluster DNS</strong><small>Routes tucked in</small></div>
            </div>
            <div className="auth-service-pill auth-service-pill--apps">
              <span><BoxesIcon /></span>
              <div><strong>Workloads</strong><small>Ready to launch</small></div>
            </div>
            <div className="auth-service-pill auth-service-pill--data">
              <span><DatabaseIcon /></span>
              <div><strong>Data</strong><small>Safe & sound</small></div>
            </div>
          </div>

          <div className="auth-story-footer" aria-label="Workspace scope">
            <span><Globe2Icon aria-hidden="true" />Multi-cluster</span>
            <span><ShieldCheckIcon aria-hidden="true" />Role locked</span>
            <span><LockKeyholeIcon aria-hidden="true" />12-hour session</span>
          </div>
        </section>
      </div>
    </main>
  )
}
