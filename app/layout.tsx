import type { Metadata } from "next"
import { Geist_Mono, Manrope } from "next/font/google"
import { headers } from "next/headers"

import { TooltipProvider } from "@/components/ui/tooltip"

import "./globals.css"

const manrope = Manrope({
  variable: "--font-manrope",
  subsets: ["latin"],
  display: "swap",
})

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
  display: "swap",
})

const title = "KubeDeck — Kubernetes Launchpad"
const description =
  "A private Kubernetes launchpad for web apps, service DNS, ports, status, and current workload run age."

export async function generateMetadata(): Promise<Metadata> {
  const requestHeaders = await headers()
  const host = requestHeaders.get("host") ?? "localhost:3000"
  const protocol = host.startsWith("localhost") ? "http" : "https"
  const baseUrl = `${protocol}://${host}`
  const socialImage = new URL("/og.png", baseUrl).toString()

  return {
    title,
    description,
    openGraph: {
      title,
      description,
      type: "website",
      url: baseUrl,
      images: [
        {
          url: socialImage,
          width: 1731,
          height: 909,
          alt: "KubeDeck Kubernetes launchpad",
        },
      ],
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
      images: [socialImage],
    },
  }
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en" className="dark">
      <body
        className={`${manrope.variable} ${geistMono.variable} antialiased`}
      >
        <TooltipProvider>{children}</TooltipProvider>
      </body>
    </html>
  )
}
