import Image from "next/image"

import { cn } from "@/lib/utils"

export function KubeDeckLogo({
  className,
  priority = false,
}: {
  className?: string
  priority?: boolean
}) {
  return (
    <span className={cn("kubedeck-logo", className)} aria-hidden="true">
      <Image
        src="/kubedeck-kb-logo.png"
        alt=""
        width={512}
        height={512}
        priority={priority}
        unoptimized
      />
    </span>
  )
}
