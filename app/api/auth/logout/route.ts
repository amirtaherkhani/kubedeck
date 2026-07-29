import { NextResponse } from "next/server"

import {
  ADMIN_SESSION_COOKIE,
  sessionCookieOptions,
} from "@/lib/auth"

export async function POST(request: Request) {
  const response = NextResponse.redirect(new URL("/", request.url), 303)
  response.cookies.set(ADMIN_SESSION_COOKIE, "", {
    ...sessionCookieOptions(),
    maxAge: 0,
  })
  return response
}
