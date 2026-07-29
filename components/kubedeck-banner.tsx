import Image from "next/image"

import { cn } from "@/lib/utils"

type KubeDeckBannerProps = {
  className?: string
  headingId: string
  priority?: boolean
}

export function KubeDeckBanner({
  className,
  headingId,
  priority = false,
}: KubeDeckBannerProps) {
  const descriptionId = `${headingId}-description`

  return (
    <section
      className={cn("kubedeck-banner", className)}
      aria-labelledby={headingId}
      aria-describedby={descriptionId}
    >
      <h1 id={headingId} className="sr-only">
        KubeDeck — Your Kubernetes ecosystem, one click away.
      </h1>
      <p id={descriptionId} className="sr-only">
        Multi-cluster access across all Kubernetes nodes and global services.
      </p>
      <Image
        className="kubedeck-banner-image"
        src="/kubedeck-banner.png"
        alt=""
        width={1731}
        height={909}
        priority={priority}
        sizes="(max-width: 900px) 100vw, 70vw"
        unoptimized
      />
    </section>
  )
}
