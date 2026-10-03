"use client"

import * as React from "react"
import { Separator as SeparatorPrimitive } from "@base-ui/react/separator"
import { motion } from "motion/react"
import type { MotionEnhancements } from "@/components/ui/motion-props"

import { cn } from "@/lib/utils"

const MotionSeparator = motion.create(SeparatorPrimitive) as unknown as React.ComponentType<SeparatorPrimitive.Props & MotionEnhancements>

function Separator({
  className,
  orientation = "horizontal",
  ...props
}: SeparatorPrimitive.Props) {
  return (
    <MotionSeparator
      data-slot="separator"
      orientation={orientation}
      initial={{ opacity: 0, scaleX: orientation === "horizontal" ? 0.85 : 1, scaleY: orientation === "vertical" ? 0.85 : 1 }}
      animate={{ opacity: 1, scaleX: 1, scaleY: 1 }}
      transition={{ duration: 0.24, ease: "easeOut" }}
      className={cn(
        "shrink-0 bg-border data-horizontal:h-px data-horizontal:w-full data-vertical:w-px data-vertical:self-stretch",
        className
      )}
      {...props}
    />
  )
}

export { Separator }
