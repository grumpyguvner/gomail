/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./webadmin/**/*.{html,js}"],
  theme: {
    extend: {
      colors: {
        'gomail': {
          50: '#f0f9ff',
          100: '#e0f2fe',
          200: '#bae6fd',
          300: '#7dd3fc',
          400: '#38bdf8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          800: '#075985',
          900: '#0c4a6e',
        },
        'health': {
          'healthy': '#10b981',
          'warning': '#f59e0b',
          'error': '#ef4444',
        }
      },
      fontFamily: {
        'sans': ['Inter', 'ui-sans-serif', 'system-ui'],
        'mono': ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular'],
      },
    },
  },
  plugins: [],
}