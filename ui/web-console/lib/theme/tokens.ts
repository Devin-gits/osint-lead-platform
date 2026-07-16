export const tokens = {
  color: {
    background: "#050816",
    surface: "#0b1224",
    surfaceElevated: "#0f172a",
    primary: "#2dd4ff",
    secondary: "#6366f1",
    success: "#34d399",
    warning: "#fbbf24",
    danger: "#f97373",
    muted: "#94a3b8",
    foreground: "#f8fafc",
    foregroundSecondary: "#cbd5e1",
    foregroundMuted: "#94a3b8",
    border: "rgba(45,212,255,0.12)",
  },
  font: {
    sans: ["var(--font-inter)", "ui-sans-serif", "system-ui", "sans-serif"],
    bodySize: "14px",
    metaSize: "12px",
    heading: {
      weight: 700,
      tracking: "-0.025em",
    },
  },
  spacing: {
    density: "compact",
    pagePad: "1.5rem",
    cardPad: "1rem",
  },
  motion: {
    fast: "150ms",
    medium: "200ms",
    easing: "cubic-bezier(0.4, 0, 0.2, 1)",
    reducedMotion: "prefers-reduced-motion",
  },
  radii: {
    card: "0.5rem",
    button: "0.375rem",
    badge: "9999px",
  },
} as const;
