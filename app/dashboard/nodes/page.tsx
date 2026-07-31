import { redirect } from "next/navigation"

import DashboardClient from "../dashboard-client"

import { getCurrentAdmin } from "@/lib/auth"

export const dynamic = "force-dynamic"

export default async function NodesPage() {
  const admin = await getCurrentAdmin()
  if (!admin) redirect("/")

  return <DashboardClient admin={admin} view="nodes" />
}
