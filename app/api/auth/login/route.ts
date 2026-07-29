import { NextResponse } from "next/server"

import {
  ADMIN_SESSION_COOKIE,
  authenticateAdmin,
  createSessionToken,
  getConfiguredAdmin,
  sessionCookieOptions,
  toPublicAdmin,
} from "@/lib/auth"

export async function POST(request: Request) {
  if (!(await getConfiguredAdmin())) {
    return NextResponse.json(
      { error: "Complete the private admin setup first." },
      { status: 409 }
    )
  }

  let payload: unknown
  try {
    payload = await request.json()
  } catch {
    return invalidCredentials()
  }

  if (!payload || typeof payload !== "object") return invalidCredentials()

  const record = payload as Record<string, unknown>
  const email = typeof record.email === "string" ? record.email : ""
  const password =
    typeof record.password === "string" ? record.password : ""
  const admin = await authenticateAdmin(email, password)

  if (!admin) return invalidCredentials()

  const response = NextResponse.json({ admin: toPublicAdmin(admin) })
  response.cookies.set(
    ADMIN_SESSION_COOKIE,
    await createSessionToken(admin),
    sessionCookieOptions()
  )
  return response
}

function invalidCredentials() {
  return NextResponse.json(
    { error: "Email or password is incorrect." },
    { status: 401 }
  )
}
