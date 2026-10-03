import type * as React from "react"
import { motion } from "motion/react"

export type MotionEnhancements = {
  initial?: object
  animate?: object
  exit?: object
  transition?: object
  whileHover?: object
  whileTap?: object
  whileFocus?: object
}

export type MotionHtmlElement<Tag extends keyof React.JSX.IntrinsicElements> =
  React.ComponentType<React.ComponentProps<Tag> & MotionEnhancements>

export const MotionDiv = motion.div as unknown as MotionHtmlElement<"div">
export const MotionSpan = motion.span as unknown as MotionHtmlElement<"span">
