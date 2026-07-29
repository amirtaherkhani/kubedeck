"use client"

import * as React from "react"
import { Radio as RadioPrimitive } from "@base-ui/react/radio"
import { RadioGroup as RadioGroupPrimitive } from "@base-ui/react/radio-group"
import { motion, type Transition } from "motion/react"

import { cn } from "@/lib/utils"

const radioSpring: Transition = {
  type: "spring",
  stiffness: 200,
  damping: 16,
}

function RadioGroup({
  className,
  ...props
}: RadioGroupPrimitive.Props<string>) {
  return (
    <RadioGroupPrimitive
      data-slot="animated-radio-group"
      className={cn("animated-radio-group", className)}
      {...props}
    />
  )
}

function Radio({
  className,
  ...props
}: Omit<RadioPrimitive.Root.Props<string>, "render">) {
  return (
    <RadioPrimitive.Root
      data-slot="animated-radio"
      className={cn("animated-radio-option", className)}
      render={(renderProps, state) => {
        const motionProps =
          renderProps as React.ComponentProps<typeof motion.span>

        return (
          <motion.span
            {...motionProps}
            initial={false}
            animate={{ y: state.checked ? -1 : 0 }}
            whileHover={{ y: -1 }}
            whileTap={{ scale: 0.96 }}
            transition={radioSpring}
          />
        )
      }}
      {...props}
    />
  )
}

function RadioIndicator({
  className,
  transition = radioSpring,
  ...props
}: Omit<RadioPrimitive.Indicator.Props, "keepMounted" | "render"> & {
  transition?: Transition
}) {
  return (
    <RadioPrimitive.Indicator
      keepMounted
      data-slot="animated-radio-indicator"
      className={cn("animated-radio-indicator", className)}
      render={(renderProps, state) => {
        const motionProps =
          renderProps as React.ComponentProps<typeof motion.span>

        return (
          <motion.span
            {...motionProps}
            initial={false}
            animate={{
              opacity: state.checked ? 1 : 0,
              scale: state.checked ? 1 : 0.35,
            }}
            transition={transition}
          />
        )
      }}
      {...props}
    />
  )
}

export { Radio, RadioGroup, RadioIndicator }
