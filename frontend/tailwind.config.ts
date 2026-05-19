import type { Config } from "tailwindcss";

export default {
  content: ["./src/**/*.{html,js,svelte,ts}"],
  theme: {
    extend: {
      colors: {
        accent: "var(--accent)",
        "accent-2": "var(--accent-2)",
        ok: "var(--ok)",
        warn: "var(--warn)",
        danger: "var(--danger)",
        "bg-0": "var(--bg-0)",
        "bg-1": "var(--bg-1)",
        "ink-900": "var(--ink-900)",
        "ink-700": "var(--ink-700)",
        "ink-500": "var(--ink-500)",
        "ink-300": "var(--ink-300)",
        surface: "var(--surface)",
      },
      fontFamily: {
        sans: ["IBM Plex Sans", "sans-serif"],
        grotesk: ["Space Grotesk", "sans-serif"],
        mono: ["IBM Plex Mono", "monospace"],
      },
      borderRadius: {
        panel: "18px",
        card: "14px",
      },
      boxShadow: {
        md: "var(--shadow-md)",
      },
    },
  },
  plugins: [],
} satisfies Config;
