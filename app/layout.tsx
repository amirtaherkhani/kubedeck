import type { Metadata } from "next"
import { Geist, Geist_Mono } from "next/font/google"
import { headers } from "next/headers"

import { TooltipProvider } from "@/components/ui/tooltip"

import "./globals.css"

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
})

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
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
          width: 1200,
          height: 630,
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
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <TooltipProvider>{children}</TooltipProvider>
      </body>
    </html>
  )
}
