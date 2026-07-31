"use client"

import * as React from "react"
import { AnimatePresence, motion } from "motion/react"
import {
  AlertCircleIcon,
  ArrowRightIcon,
  AtSignIcon,
  CheckIcon,
  EyeIcon,
  EyeOffIcon,
  KeyRoundIcon,
  LoaderCircleIcon,
  ShieldCheckIcon,
  UserRoundIcon,
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
  const [showPassword, setShowPassword] = React.useState(false)
  const [showPasswordConfirmation, setShowPasswordConfirmation] =
    React.useState(false)
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
    <form className="auth-form" onSubmit={onSubmit}>
      <FieldSet disabled={pending}>
        <FieldGroup className="auth-field-group">
          {isSetup && (
            <div className="auth-name-grid">
              <Field>
                <FieldLabel htmlFor="firstName">First name</FieldLabel>
                <div className="auth-input-shell">
                  <UserRoundIcon aria-hidden="true" />
                  <Input
                    id="firstName"
                    name="firstName"
                    autoComplete="given-name"
                    maxLength={80}
                    placeholder="Avery"
                    autoFocus
                    required
                  />
                </div>
              </Field>
              <Field>
                <FieldLabel htmlFor="lastName">Last name</FieldLabel>
                <div className="auth-input-shell">
                  <UserRoundIcon aria-hidden="true" />
                  <Input
                    id="lastName"
                    name="lastName"
                    autoComplete="family-name"
                    maxLength={80}
                    placeholder="Morgan"
                    required
                  />
                </div>
              </Field>
            </div>
          )}

          <Field>
            <FieldLabel htmlFor="email">Admin email</FieldLabel>
            <div className="auth-input-shell">
              <AtSignIcon aria-hidden="true" />
              <Input
                id="email"
                name="email"
                type="email"
                autoComplete="username"
                inputMode="email"
                maxLength={254}
                placeholder="admin@your-cluster.dev"
                spellCheck={false}
                autoFocus={!isSetup}
                required
              />
            </div>
          </Field>

          <Field>
            <FieldLabel htmlFor="password">Password</FieldLabel>
            <div className="auth-input-shell auth-input-shell--password">
              <KeyRoundIcon aria-hidden="true" />
              <Input
                id="password"
                name="password"
                type={showPassword ? "text" : "password"}
                autoComplete={isSetup ? "new-password" : "current-password"}
                minLength={isSetup ? 12 : undefined}
                maxLength={128}
                placeholder={isSetup ? "Create a strong password" : "Enter your password"}
                required
              />
              <button
                className="auth-password-toggle"
                type="button"
                aria-label={showPassword ? "Hide password" : "Show password"}
                aria-pressed={showPassword}
                onClick={() => setShowPassword((visible) => !visible)}
              >
                {showPassword ? (
                  <EyeOffIcon aria-hidden="true" />
                ) : (
                  <EyeIcon aria-hidden="true" />
                )}
              </button>
            </div>
            {isSetup && (
              <FieldDescription className="auth-password-hint">
                <span>
                  <CheckIcon aria-hidden="true" />
                  12–128 characters
                </span>
                <span>
                  <CheckIcon aria-hidden="true" />
                  Salted hash only
                </span>
              </FieldDescription>
            )}
          </Field>

          {isSetup && (
            <Field data-invalid={Boolean(confirmError)}>
              <FieldLabel htmlFor="passwordConfirmation">
                Confirm password
              </FieldLabel>
              <div className="auth-input-shell auth-input-shell--password">
                <KeyRoundIcon aria-hidden="true" />
                <Input
                  id="passwordConfirmation"
                  name="passwordConfirmation"
                  type={showPasswordConfirmation ? "text" : "password"}
                  autoComplete="new-password"
                  minLength={12}
                  maxLength={128}
                  placeholder="Repeat your password"
                  aria-invalid={Boolean(confirmError)}
                  required
                />
                <button
                  className="auth-password-toggle"
                  type="button"
                  aria-label={
                    showPasswordConfirmation
                      ? "Hide password confirmation"
                      : "Show password confirmation"
                  }
                  aria-pressed={showPasswordConfirmation}
                  onClick={() =>
                    setShowPasswordConfirmation((visible) => !visible)
                  }
                >
                  {showPasswordConfirmation ? (
                    <EyeOffIcon aria-hidden="true" />
                  ) : (
                    <EyeIcon aria-hidden="true" />
                  )}
                </button>
              </div>
              <FieldError>{confirmError}</FieldError>
            </Field>
          )}

          <AnimatePresence initial={false}>
            {error && (
              <motion.div
                initial={{ opacity: 0, height: 0, y: -6 }}
                animate={{ opacity: 1, height: "auto", y: 0 }}
                exit={{ opacity: 0, height: 0, y: -6 }}
              >
                <Alert variant="destructive">
                  <AlertCircleIcon />
                  <AlertTitle>Unable to continue</AlertTitle>
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              </motion.div>
            )}
          </AnimatePresence>

          <Button
            type="submit"
            size="lg"
            className="auth-submit-button w-full"
            disabled={pending}
          >
            <AnimatePresence mode="wait" initial={false}>
              <motion.span
                key={pending ? "pending" : mode}
                className="inline-flex items-center gap-2"
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -4 }}
              >
                {pending ? (
                  <>
                    <LoaderCircleIcon
                      className="animate-spin"
                      data-icon="inline-start"
                    />
                    {isSetup ? "Creating your deck..." : "Opening your deck..."}
                  </>
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
              </motion.span>
            </AnimatePresence>
          </Button>
        </FieldGroup>
      </FieldSet>
    </form>
  )
}
