import { redirect } from "next/navigation"

import SettingsClient from "./settings-client"

import { getCurrentAdmin } from "@/lib/auth"

export const dynamic = "force-dynamic"

export default async function SettingsPage() {
  const admin = await getCurrentAdmin()
  if (!admin) redirect("/")

  return <SettingsClient admin={admin} />
}
