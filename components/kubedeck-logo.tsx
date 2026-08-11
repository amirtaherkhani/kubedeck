import { useId } from "react"

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
  const id = useId().replaceAll(":", "")
  const surfaceGradient = `${id}-surface`
  const markGradient = `${id}-mark`

  return (
    <span
      className={cn("kubedeck-logo", `kubedeck-logo--${variant}`, className)}
      data-logo-priority={priority ? "high" : "auto"}
      data-logo-variant={variant}
      aria-hidden="true"
    >
      <svg
        viewBox="0 0 1024 1024"
        role="presentation"
        focusable="false"
        xmlns="http://www.w3.org/2000/svg"
      >
        <defs>
          <linearGradient id={surfaceGradient} x1="120" y1="84" x2="900" y2="940">
            <stop className="kubedeck-logo__surface-start" offset="0" />
            <stop className="kubedeck-logo__surface-end" offset="1" />
          </linearGradient>
          <linearGradient id={markGradient} x1="168" y1="168" x2="840" y2="826">
            <stop className="kubedeck-logo__mark-start" offset="0" />
            <stop className="kubedeck-logo__mark-end" offset="1" />
          </linearGradient>
          <filter id={`${id}-shadow`} x="-20%" y="-20%" width="140%" height="140%">
            <feDropShadow
              dx="0"
              dy="20"
              stdDeviation="24"
              className="kubedeck-logo__shadow"
            />
          </filter>
        </defs>

        <rect
          x="48"
          y="48"
          width="928"
          height="928"
          rx="248"
          fill={`url(#${surfaceGradient})`}
          filter={`url(#${id}-shadow)`}
          className="kubedeck-logo__surface"
        />
        <path
          d="M258 92h508c91 0 166 75 166 166v508c0 91-75 166-166 166H258c-91 0-166-75-166-166V258c0-91 75-166 166-166Z"
          fill="none"
          className="kubedeck-logo__edge"
        />
        <path
          d="M246 116h416c138 0 250 112 250 250v22C675 266 486 176 264 176h-30c-20 0-30-24-16-38l28-22Z"
          className="kubedeck-logo__specular"
        />
        <path
          d="M482 132 176 316Q160 326 160 344v336q0 18 16 28l306 184V742L282 622v-70l162 162V586l-88-88 126-90V300L282 420v-64l200-120Z"
          fill={`url(#${markGradient})`}
          className="kubedeck-logo__mark"
        />
        <path
          fillRule="evenodd"
          d="M510 132 832 326q16 10 16 28v98l-80 48 80 48v132q0 18-16 28L510 892Zm94 198v198l126-76v-46Zm0 212v198l126-76v-46Z"
          fill={`url(#${markGradient})`}
          className="kubedeck-logo__mark"
        />
        <path
          d="m204 318 276-164v72L248 358v62l-34 20c-14 8-30-2-30-18v-76c0-12 7-22 20-28Z"
          className="kubedeck-logo__mark-highlight"
        />
      </svg>
    </span>
  )
}
