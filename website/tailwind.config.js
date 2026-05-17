/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        accent: '#caff00',
        danger: '#ff3333',
        success: '#00ff88',
        surface: '#0f0f0f',
        border: '#1a1a1a',
      },
      fontFamily: {
        bebas: ['"Bebas Neue"', 'cursive'],
        mono: ['"Courier New"', 'Courier', 'monospace'],
      },
    },
  },
  plugins: [],
}
