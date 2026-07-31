import { notFound, redirect } from "next/navigation"

import DashboardClient from "../../dashboard-client"

import { getCurrentAdmin } from "@/lib/auth"

export const dynamic = "force-dynamic"

const categoryIds = new Set([
  "web-applications",
  "databases-storage",
  "observability-metrics",
  "automation-workflows",
  "deployments",
  "ai-services",
  "messaging-events",
  "developer-tools",
  "platform-security",
])

export default async function CatalogCategoryPage({
  params,
}: {
  params: Promise<{ category: string }>
}) {
  const [admin, { category }] = await Promise.all([
    getCurrentAdmin(),
    params,
  ])
  if (!admin) redirect("/")
  if (!categoryIds.has(category)) notFound()

  return (
    <DashboardClient
      admin={admin}
      view="catalog"
      catalogCategory={category}
    />
  )
}
