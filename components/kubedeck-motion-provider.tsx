"use client"

import * as React from "react"
import { MotionConfig } from "motion/react"

export function KubeDeckMotionProvider({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <MotionConfig
      reducedMotion="user"
      transition={{ type: "spring", stiffness: 260, damping: 24 }}
    >
      {children}
    </MotionConfig>
  )
}
