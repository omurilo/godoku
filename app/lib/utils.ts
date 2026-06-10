type ClassArg = string | false | null | undefined;

/**
 * cn joins truthy class names with a space. The shell uses hand-written CSS
 * (no Tailwind), so a plain join is all we need — and it keeps the bundle free
 * of clsx/tailwind-merge.
 */
export function cn(...args: ClassArg[]): string {
  return args.filter(Boolean).join(" ");
}
