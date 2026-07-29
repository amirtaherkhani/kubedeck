import { NextResponse } from "next/server"

import {
  ADMIN_SESSION_COOKIE,
  createAdmin,
  createSessionToken,
  getConfiguredAdmin,
  sessionCookieOptions,
  toPublicAdmin,
  validateAdminInput,
} from "@/lib/auth"

export async function POST(request: Request) {
  if (!request.headers.get("oai-authenticated-user-email")) {
    return NextResponse.json(
      { error: "Private workspace verification is required for first setup." },
      { status: 403 }
    )
  }

  if (await getConfiguredAdmin()) {
    return NextResponse.json(
      { error: "The KubeDeck admin account is already configured." },
      { status: 409 }
    )
  }

  let payload: unknown
  try {
    payload = await request.json()
  } catch {
    return NextResponse.json(
      { error: "Enter all required admin details." },
      { status: 400 }
    )
  }

  const validated = validateAdminInput(payload)
  if (!validated.ok) {
    return NextResponse.json(
      { error: validated.error },
      { status: 400 }
    )
  }

  const admin = await createAdmin(validated.data)
  if (!admin) {
    return NextResponse.json(
      { error: "The KubeDeck admin account is already configured." },
      { status: 409 }
    )
  }

  const response = NextResponse.json(
    { admin: toPublicAdmin(admin) },
    { status: 201 }
  )
  response.cookies.set(
    ADMIN_SESSION_COOKIE,
    await createSessionToken(admin),
    sessionCookieOptions()
  )
  return response
}
