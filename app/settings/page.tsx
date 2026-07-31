import { redirect } from "next/navigation"

import SettingsClient from "./settings-client"

import { getCurrentAdmin } from "@/lib/auth"
import { readInitialClusterSnapshot } from "@/lib/kubedeck-agent"

export const dynamic = "force-dynamic"

export default async function SettingsPage() {
  const admin = await getCurrentAdmin()
  if (!admin) redirect("/")

  const initialSnapshot = await readInitialClusterSnapshot()

  return <SettingsClient admin={admin} initialSnapshot={initialSnapshot} />
}
