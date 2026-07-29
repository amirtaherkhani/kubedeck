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
        src="/brand/kubedeck-mark.svg"
        alt=""
        width={1024}
        height={1024}
        priority={priority}
        unoptimized
      />
    </span>
  )
}
