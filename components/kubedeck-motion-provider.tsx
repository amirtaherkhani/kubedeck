"use client"

import * as React from "react"
import { MotionConfig } from "motion/react"

export function KubeDeckMotionProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [motionEnabled, setMotionEnabled] = React.useState(true)

  React.useEffect(() => {
    function readMotionPreference() {
      const enabled =
        window.localStorage.getItem("kubedeck-motion-enabled") !== "false"
      setMotionEnabled(enabled)
      document.documentElement.classList.toggle(
        "kubedeck-motion-off",
        !enabled
      )
    }

    readMotionPreference()
    window.addEventListener("kubedeck:preferences", readMotionPreference)
    return () =>
      window.removeEventListener("kubedeck:preferences", readMotionPreference)
  }, [])

  return (
    <MotionConfig
      reducedMotion={motionEnabled ? "user" : "always"}
      transition={{ type: "spring", stiffness: 260, damping: 24 }}
    >
      {children}
    </MotionConfig>
  )
}
