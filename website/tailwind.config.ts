import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: "class",
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        fortress: {
          navy: "#0A1628",
          "navy-light": "#1A2B45",
          green: "#0F6E56",
          "green-light": "#1D9E75",
          amber: "#EF9F27",
          gray: "#8892A4",
          "gray-light": "#B4BCC9",
        },
      },
      animation: {
        "blink": "blink 1s step-end infinite",
        "typing": "typing 3s steps(40, end)",
        "fade-in-up": "fade-in-up 0.6s ease-out forwards",
        "count-up": "count-up 2s ease-out forwards",
      },
      keyframes: {
        blink: {
          "0%, 100%": { opacity: "1" },
          "50%": { opacity: "0" },
        },
        typing: {
          "from": { width: "0" },
          "to": { width: "100%" },
        },
        "fade-in-up": {
          "0%": { opacity: "0", transform: "translateY(20px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        "count-up": {
          "0%": { opacity: "0", transform: "translateY(10px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
      },
    },
  },
  plugins: [],
};
export default config;
