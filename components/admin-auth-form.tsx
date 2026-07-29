"use client"

import * as React from "react"
import {
  AlertCircleIcon,
  ArrowRightIcon,
  ShieldCheckIcon,
} from "lucide-react"

import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type AdminAuthFormProps = {
  mode: "setup" | "login"
}

export function AdminAuthForm({ mode }: AdminAuthFormProps) {
  const [pending, setPending] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [confirmError, setConfirmError] = React.useState<string | null>(null)
  const isSetup = mode === "setup"

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)
    setConfirmError(null)

    const form = event.currentTarget
    const formData = new FormData(form)
    const password = String(formData.get("password") ?? "")
    const passwordConfirmation = String(
      formData.get("passwordConfirmation") ?? ""
    )

    if (isSetup && password !== passwordConfirmation) {
      setConfirmError("Passwords do not match.")
      return
    }

    setPending(true)
    try {
      const payload = isSetup
        ? {
            firstName: String(formData.get("firstName") ?? ""),
            lastName: String(formData.get("lastName") ?? ""),
            email: String(formData.get("email") ?? ""),
            password,
          }
        : {
            email: String(formData.get("email") ?? ""),
            password,
          }
      const response = await fetch(
        isSetup ? "/api/auth/setup" : "/api/auth/login",
        {
          method: "POST",
          headers: { "content-type": "application/json" },
          body: JSON.stringify(payload),
        }
      )
      const result = (await response.json().catch(() => null)) as {
        error?: string
      } | null

      if (!response.ok) {
        setError(result?.error ?? "Unable to continue. Please try again.")
        return
      }

      window.location.replace("/dashboard")
    } catch {
      setError("Unable to reach KubeDeck. Please try again.")
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={onSubmit}>
      <FieldSet disabled={pending}>
        <FieldGroup>
          {isSetup && (
            <>
              <Field>
                <FieldLabel htmlFor="firstName">First name</FieldLabel>
                <Input
                  id="firstName"
                  name="firstName"
                  autoComplete="given-name"
                  maxLength={80}
                  required
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="lastName">Last name</FieldLabel>
                <Input
                  id="lastName"
                  name="lastName"
                  autoComplete="family-name"
                  maxLength={80}
                  required
                />
              </Field>
            </>
          )}

          <Field>
            <FieldLabel htmlFor="email">Admin email</FieldLabel>
            <Input
              id="email"
              name="email"
              type="email"
              autoComplete="username"
              inputMode="email"
              maxLength={254}
              spellCheck={false}
              required
            />
          </Field>

          <Field>
            <FieldLabel htmlFor="password">Password</FieldLabel>
            <Input
              id="password"
              name="password"
              type="password"
              autoComplete={isSetup ? "new-password" : "current-password"}
              minLength={isSetup ? 12 : undefined}
              maxLength={128}
              required
            />
            {isSetup && (
              <FieldDescription>
                Use at least 12 characters. KubeDeck stores only a salted hash.
              </FieldDescription>
            )}
          </Field>

          {isSetup && (
            <Field data-invalid={Boolean(confirmError)}>
              <FieldLabel htmlFor="passwordConfirmation">
                Confirm password
              </FieldLabel>
              <Input
                id="passwordConfirmation"
                name="passwordConfirmation"
                type="password"
                autoComplete="new-password"
                minLength={12}
                maxLength={128}
                aria-invalid={Boolean(confirmError)}
                required
              />
              <FieldError>{confirmError}</FieldError>
            </Field>
          )}

          {error && (
            <Alert variant="destructive">
              <AlertCircleIcon />
              <AlertTitle>Unable to continue</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <Button
            type="submit"
            size="lg"
            className="w-full"
            disabled={pending}
          >
            {pending ? (
              "Securing account..."
            ) : isSetup ? (
              <>
                <ShieldCheckIcon data-icon="inline-start" />
                Create admin account
              </>
            ) : (
              <>
                Sign in
                <ArrowRightIcon data-icon="inline-end" />
              </>
            )}
          </Button>
        </FieldGroup>
      </FieldSet>
    </form>
  )
}
