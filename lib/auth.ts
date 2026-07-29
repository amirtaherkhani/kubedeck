import { env as cloudflareEnv } from "cloudflare:workers"
import { cookies } from "next/headers"

import { getD1 } from "@/db"

export const ADMIN_SESSION_COOKIE = "__Host-kubedeck_admin"
export const ADMIN_SESSION_MAX_AGE = 12 * 60 * 60
export const PASSWORD_MIN_LENGTH = 12
export const ADMIN_ENV_KEYS = {
  firstName: "KUBEDECK_ADMIN_FIRST_NAME",
  lastName: "KUBEDECK_ADMIN_LAST_NAME",
  email: "KUBEDECK_ADMIN_EMAIL",
  password: "KUBEDECK_ADMIN_PASSWORD",
} as const

const ADMIN_ID = 1
const PASSWORD_ITERATIONS = 210_000
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/u
const encoder = new TextEncoder()
const reportedBootstrapWarnings = new Set<string>()

export type AdminUser = {
  id: number
  firstName: string
  lastName: string
  email: string
  passwordHash: string
  passwordSalt: string
  passwordIterations: number
  sessionSecret: string
  createdAt: string
  updatedAt: string
}

export type PublicAdmin = Pick<
  AdminUser,
  "id" | "firstName" | "lastName" | "email"
> & {
  role: "admin"
}

export type AdminInput = {
  firstName: string
  lastName: string
  email: string
  password: string
}

type SessionPayload = {
  adminId: number
  email: string
  issuedAt: number
  expiresAt: number
}

export function normalizeEmail(value: string) {
  return value.trim().toLowerCase()
}

export function validateAdminInput(payload: unknown):
  | { ok: true; data: AdminInput }
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

export function toPublicAdmin(admin: AdminUser): PublicAdmin {
  return {
    id: admin.id,
    firstName: admin.firstName,
    lastName: admin.lastName,
    email: admin.email,
    role: "admin",
  }
}

export async function getAdmin(): Promise<AdminUser | null> {
  return getD1()
    .prepare(
      `SELECT
        id,
        first_name AS firstName,
        last_name AS lastName,
        email,
        password_hash AS passwordHash,
        password_salt AS passwordSalt,
        password_iterations AS passwordIterations,
        session_secret AS sessionSecret,
        created_at AS createdAt,
        updated_at AS updatedAt
      FROM admin_users
      WHERE id = ?
      LIMIT 1`
    )
    .bind(ADMIN_ID)
    .first<AdminUser>()
}

export async function getConfiguredAdmin(): Promise<AdminUser | null> {
  const existingAdmin = await getAdmin()
  if (existingAdmin) return existingAdmin

  const environmentInput = getEnvironmentAdminInput()
  if (!environmentInput) return null

  const createdAdmin = await createAdmin(environmentInput)
  return createdAdmin ?? getAdmin()
}

export async function createAdmin(
  input: AdminInput
): Promise<AdminUser | null> {
  const salt = randomBytes(16)
  const passwordHash = await derivePasswordHash(
    input.password,
    salt,
    PASSWORD_ITERATIONS
  )
  const sessionSecret = toBase64Url(randomBytes(32))

  try {
    const result = await getD1()
      .prepare(
        `INSERT INTO admin_users (
          id,
          first_name,
          last_name,
          email,
          password_hash,
          password_salt,
          password_iterations,
          session_secret
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .bind(
        ADMIN_ID,
        input.firstName.trim(),
        input.lastName.trim(),
        normalizeEmail(input.email),
        passwordHash,
        toBase64Url(salt),
        PASSWORD_ITERATIONS,
        sessionSecret
      )
      .run()

    if (result.meta.changes !== 1) return null
    return getAdmin()
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    if (
      message.includes("UNIQUE constraint failed") ||
      message.includes("PRIMARY KEY")
    ) {
      return null
    }
    throw error
  }
}

export async function authenticateAdmin(
  email: string,
  password: string
): Promise<AdminUser | null> {
  const admin = await getConfiguredAdmin()
  if (!admin) return null

  const emailMatches = constantTimeTextEqual(
    normalizeEmail(email),
    admin.email
  )
  const passwordMatches = await verifyPassword(password, admin)

  return emailMatches && passwordMatches ? admin : null
}

export async function createSessionToken(admin: AdminUser) {
  const now = Math.floor(Date.now() / 1000)
  const payload: SessionPayload = {
    adminId: admin.id,
    email: admin.email,
    issuedAt: now,
    expiresAt: now + ADMIN_SESSION_MAX_AGE,
  }
  const encodedPayload = toBase64Url(encoder.encode(JSON.stringify(payload)))
  const signature = await sign(encodedPayload, admin.sessionSecret)

  return `${encodedPayload}.${signature}`
}

export async function getAdminFromSessionToken(
  token: string | undefined
): Promise<PublicAdmin | null> {
  if (!token) return null

  const admin = await getAdmin()
  if (!admin) return null

  const [encodedPayload, signature, extra] = token.split(".")
  if (!encodedPayload || !signature || extra) return null

  const validSignature = await verifySignature(
    encodedPayload,
    signature,
    admin.sessionSecret
  )
  if (!validSignature) return null

  let payload: SessionPayload
  try {
    payload = JSON.parse(
      new TextDecoder().decode(fromBase64Url(encodedPayload))
    ) as SessionPayload
  } catch {
    return null
  }

  const now = Math.floor(Date.now() / 1000)
  if (
    payload.adminId !== admin.id ||
    payload.email !== admin.email ||
    !Number.isInteger(payload.issuedAt) ||
    !Number.isInteger(payload.expiresAt) ||
    payload.issuedAt > now + 60 ||
    payload.expiresAt <= now
  ) {
    return null
  }

  return toPublicAdmin(admin)
}

export async function getCurrentAdmin(): Promise<PublicAdmin | null> {
  const cookieStore = await cookies()
  return getAdminFromSessionToken(
    cookieStore.get(ADMIN_SESSION_COOKIE)?.value
  )
}

export function sessionCookieOptions() {
  return {
    httpOnly: true,
    secure: true,
    sameSite: "strict" as const,
    path: "/",
    maxAge: ADMIN_SESSION_MAX_AGE,
  }
}

function getEnvironmentAdminInput(): AdminInput | null {
  const runtimeEnvironment = cloudflareEnv as unknown as Record<
    string,
    unknown
  >
  const values = Object.fromEntries(
    Object.entries(ADMIN_ENV_KEYS).map(([field, key]) => [
      field,
      typeof runtimeEnvironment[key] === "string"
        ? runtimeEnvironment[key]
        : "",
    ])
  ) as AdminInput
  const configuredFields = Object.entries(values).filter(
    ([, value]) => value.length > 0
  )

  if (configuredFields.length === 0) return null

  if (configuredFields.length !== Object.keys(ADMIN_ENV_KEYS).length) {
    const missingKeys = Object.entries(ADMIN_ENV_KEYS)
      .filter(([field]) => !values[field as keyof AdminInput])
      .map(([, key]) => key)
    reportBootstrapWarning(
      `Admin environment bootstrap is incomplete. Missing: ${missingKeys.join(", ")}.`
    )
    return null
  }

  const validated = validateAdminInput(values)
  if (!validated.ok) {
    reportBootstrapWarning(
      `Admin environment bootstrap was skipped: ${validated.error}`
    )
    return null
  }

  return validated.data
}

function reportBootstrapWarning(message: string) {
  if (reportedBootstrapWarnings.has(message)) return
  reportedBootstrapWarnings.add(message)
  console.warn(`[KubeDeck] ${message}`)
}

async function verifyPassword(password: string, admin: AdminUser) {
  const actual = await derivePasswordHash(
    password,
    fromBase64Url(admin.passwordSalt),
    admin.passwordIterations
  )
  return constantTimeTextEqual(actual, admin.passwordHash)
}

async function derivePasswordHash(
  password: string,
  salt: Uint8Array,
  iterations: number
) {
  const saltBytes = new Uint8Array(salt.byteLength)
  saltBytes.set(salt)
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    encoder.encode(password),
    "PBKDF2",
    false,
    ["deriveBits"]
  )
  const bits = await crypto.subtle.deriveBits(
    {
      name: "PBKDF2",
      hash: "SHA-256",
      salt: saltBytes.buffer,
      iterations,
    },
    keyMaterial,
    256
  )

  return toBase64Url(new Uint8Array(bits))
}

async function sign(value: string, secret: string) {
  const key = await importHmacKey(secret, ["sign"])
  const signature = await crypto.subtle.sign(
    "HMAC",
    key,
    encoder.encode(value)
  )
  return toBase64Url(new Uint8Array(signature))
}

async function verifySignature(
  value: string,
  signature: string,
  secret: string
) {
  try {
    const key = await importHmacKey(secret, ["verify"])
    return crypto.subtle.verify(
      "HMAC",
      key,
      fromBase64Url(signature),
      encoder.encode(value)
    )
  } catch {
    return false
  }
}

function importHmacKey(secret: string, usages: KeyUsage[]) {
  return crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    usages
  )
}

function randomBytes(length: number) {
  const bytes = new Uint8Array(length)
  crypto.getRandomValues(bytes)
  return bytes
}

function toBase64Url(bytes: Uint8Array) {
  let binary = ""
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
    .replaceAll("+", "-")
    .replaceAll("/", "_")
    .replace(/=+$/u, "")
}

function fromBase64Url(value: string) {
  const normalized = value.replaceAll("-", "+").replaceAll("_", "/")
  const padded = normalized.padEnd(
    normalized.length + ((4 - (normalized.length % 4)) % 4),
    "="
  )
  const binary = atob(padded)
  return Uint8Array.from(binary, (character) => character.charCodeAt(0))
}

function constantTimeTextEqual(left: string, right: string) {
  const leftBytes = encoder.encode(left)
  const rightBytes = encoder.encode(right)
  const maxLength = Math.max(leftBytes.length, rightBytes.length)
  let difference = leftBytes.length ^ rightBytes.length

  for (let index = 0; index < maxLength; index += 1) {
    difference |=
      (leftBytes[index] ?? 0) ^ (rightBytes[index] ?? 0)
  }

  return difference === 0
}
