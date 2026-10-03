import { cn } from "@/lib/utils"

export type KubeDeckLogoVariant = "light" | "dark" | "clear"

export function KubeDeckLogo({
  className,
  priority = false,
  variant = "dark",
}: {
  className?: string
  priority?: boolean
  variant?: KubeDeckLogoVariant
}) {
  return (
    <span
      className={cn("kubedeck-logo", `kubedeck-logo--${variant}`, className)}
      data-logo-priority={priority ? "high" : "auto"}
      data-logo-variant={variant}
      aria-hidden="true"
    >
      <svg
        viewBox="0 0 256 256"
        role="presentation"
        focusable="false"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          className="kubedeck-logo__cube"
          d="m128 34 78 45v90l-78 45-78-45V79z"
          fill="none"
          strokeWidth="20"
          strokeLinejoin="round"
        />
        <path
          className="kubedeck-logo__letter"
          d="M91 88v80m4-39 57-42m-57 42 61 44"
          fill="none"
          strokeWidth="22"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <circle className="kubedeck-logo__node" cx="190" cy="128" r="10" />
      </svg>
    </span>
  )
}
