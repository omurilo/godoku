import React from "react";
import { cn } from "../lib/utils";
import type { BannerData } from "../types";

/**
 * Banner renders the optional site-wide announcement from godoku.yaml. It is
 * server-rendered (visible by default), and — when dismissible — hidden after
 * mount if the user has already closed this exact message (remembered in
 * localStorage, keyed by the message so a new announcement reappears).
 */
export function Banner({ banner }: { banner: BannerData }) {
  const [dismissed, setDismissed] = React.useState(false);
  const key = "godoku-banner";

  React.useEffect(() => {
    if (banner.dismissible && window.localStorage.getItem(key) === banner.message) {
      setDismissed(true);
    }
  }, [banner.message, banner.dismissible]);

  if (dismissed) return null;

  const dismiss = () => {
    window.localStorage.setItem(key, banner.message);
    setDismissed(true);
  };

  return (
    <div className={cn("gd-banner", "gd-banner--" + (banner.color || "info"))}>
      <span className="gd-banner-msg">{banner.message}</span>
      {banner.dismissible ? (
        <button type="button" className="gd-banner-close" onClick={dismiss} aria-label="Dismiss">
          ✕
        </button>
      ) : null}
    </div>
  );
}
