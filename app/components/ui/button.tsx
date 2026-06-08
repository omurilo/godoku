import React from "react";
import { cn } from "../../lib/utils";

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary";
  size?: "sm";
  block?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "primary", size = "sm", block, ...props }, ref) => {
    return (
      <button
        ref={ref}
        className={cn(
          "gd-btn",
          variant === "primary" && "gd-btn--primary",
          size === "sm" && "gd-btn--sm",
          block && "gd-btn--block",
          className,
        )}
        {...props}
      />
    );
  },
);
Button.displayName = "Button";
