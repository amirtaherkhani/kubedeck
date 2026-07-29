import type { MetadataRoute } from "next"

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "KubeDeck",
    short_name: "KubeDeck",
    description:
      "A private multi-cluster Kubernetes launchpad for services, DNS, status, and workload health.",
    start_url: "/dashboard",
    display: "standalone",
    background_color: "#020b18",
    theme_color: "#061425",
    icons: [
      {
        src: "/brand/kubedeck-mark-192.png",
        sizes: "192x192",
        type: "image/png",
      },
      {
        src: "/brand/kubedeck-mark-512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "any maskable",
      },
      {
        src: "/brand/kubedeck-mark.svg",
        sizes: "any",
        type: "image/svg+xml",
        purpose: "any",
      },
    ],
  }
}
