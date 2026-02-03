
export default {
  darkMode: 'class',
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        "bg-light": "#f8fafc",
        "bg-dark": "#09090b",
        "surface-light": "#ffffff",
        "surface-dark": "#121215",
        "primary": "#3b82f6",
        "text-main-light": "#0f172a",
        "text-sub-light": "#64748b",
        "text-main-dark": "#f8fafc",
        "text-sub-dark": "#a1a1aa",
      },
      fontFamily: {
        heading: ['Outfit', 'sans-serif'],
      }
    },
  },
  plugins: [],
}
