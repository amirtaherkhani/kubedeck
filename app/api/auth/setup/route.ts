import { NextResponse } from "next/server"

import {
  ADMIN_SESSION_COOKIE,
  PASSWORD_MIN_LENGTH,
  createAdmin,
  createSessionToken,
  getAdmin,
  normalizeEmail,
  sessionCookieOptions,
  toPublicAdmin,
} from "@/lib/auth"

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/u

export async function POST(request: Request) {
  if (!request.headers.get("oai-authenticated-user-email")) {
    return NextResponse.json(
      { error: "Private workspace verification is required for first setup." },
      { status: 403 }
    )
  }

  if (await getAdmin()) {
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

  const validated = validateSetupPayload(payload)
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

function validateSetupPayload(payload: unknown):
  | {
      ok: true
      data: {
        firstName: string
        lastName: string
        email: string
        password: string
      }
    }
  | { ok: false; error: string } {
  if (!payload || typeof payload !== "object") {
    return { ok: false, error: "Enter all required admin details." }
  }

  const record = payload as Record<string, unknown>
  const firstName =
    typeof record.firstName === "string" ? record.firstName.trim() : ""
  const lastName =
    typeof record.lastName === "string" ? record.lastName.trim() : ""
  const email =
    typeof record.email === "string" ? normalizeEmail(record.email) : ""
  const password =
    typeof record.password === "string" ? record.password : ""

  if (!firstName || !lastName) {
    return { ok: false, error: "First name and last name are required." }
  }
  if (firstName.length > 80 || lastName.length > 80) {
    return { ok: false, error: "Names must be 80 characters or fewer." }
  }
  if (!EMAIL_PATTERN.test(email) || email.length > 254) {
    return { ok: false, error: "Enter a valid admin email address." }
  }
  if (password.length < PASSWORD_MIN_LENGTH || password.length > 128) {
    return {
      ok: false,
      error: `Password must be between ${PASSWORD_MIN_LENGTH} and 128 characters.`,
    }
  }
  if (password.trim().toLowerCase() === email) {
    return { ok: false, error: "Password cannot match the admin email." }
  }

  return {
    ok: true,
    data: { firstName, lastName, email, password },
  }
}
