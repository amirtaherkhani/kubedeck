import { redirect } from "next/navigation"

import DashboardClient from "../dashboard-client"

import { getCurrentAdmin } from "@/lib/auth"
import { readInitialClusterSnapshot } from "@/lib/kubedeck-agent"

export const dynamic = "force-dynamic"

export default async function NodesPage() {
  const admin = await getCurrentAdmin()
  if (!admin) redirect("/")

  const initialSnapshot = await readInitialClusterSnapshot()

  return (
    <DashboardClient
      admin={admin}
      view="nodes"
      initialSnapshot={initialSnapshot}
    />
  )
}
