import Image from "next/image"

import { cn } from "@/lib/utils"

type KubeDeckBannerProps = {
  className?: string
  headingId: string
  liveGraph?: boolean
  priority?: boolean
}

const liveGraphLinks = [
  "top",
  "left",
  "right",
  "lower-left",
  "lower-right",
  "bottom",
] as const

const liveGraphBeacons = [
  "core",
  "top",
  "left",
  "right",
  "lower-left",
  "lower-right",
  "bottom",
] as const

export function KubeDeckBanner({
  className,
  headingId,
  liveGraph = false,
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
      {liveGraph ? (
        <div className="kubedeck-live-graph" aria-hidden="true">
          {liveGraphLinks.map((link) => (
            <span
              key={link}
              className={`kubedeck-live-link kubedeck-live-link--${link}`}
            />
          ))}
          {liveGraphBeacons.map((beacon) => (
            <span
              key={beacon}
              className={`kubedeck-live-beacon kubedeck-live-beacon--${beacon}`}
            />
          ))}
        </div>
      ) : null}
    </section>
  )
}
