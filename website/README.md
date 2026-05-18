# HAVOC — Portfolio Website

Interactive portfolio showcase for [Havoc](../README.md), a production-depth Kubernetes chaos engineering platform. Built with Vite + React, Tailwind CSS, Framer Motion, and Recharts.

## What's inside

| Section | Description |
|---|---|
| **Hero** | Animated node-grid background simulating a pod kill cycle |
| **Architecture** | Live data-flow diagram with animated Kafka packet arrows; click any component for details |
| **Demo** | Interactive experiment control panel — configure, run, and watch guardrail checks and pod state transitions |
| **Guardrails** | Four interactive safety cards: blast radius slider, Redis lock animation, kill switch, and blackout window calendar |
| **Capabilities** | Chaos action cards, full tech stack table, and deployment target cards |
| **About** | Project description, stats, and contact links |

## Run locally

```bash
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173).

## Build

```bash
npm run build      # outputs to dist/
npm run preview    # preview the production build locally
```

## Deploy to Vercel

### One-click (recommended)

1. Push this repo to GitHub
2. Go to [vercel.com](https://vercel.com) → **Add New Project**
3. Import your repo, set **Root Directory** to `website`
4. Vercel auto-detects Vite — click **Deploy**

`vercel.json` is already configured to handle client-side routing.

### Vercel CLI

```bash
npm i -g vercel
cd website
vercel --prod
```

## Tech stack

- **Vite** + **React** (functional components, hooks)
- **Tailwind CSS v3** with custom design tokens
- **Framer Motion** for scroll-triggered and entrance animations
- **Recharts** (available, dark-themed)
- Zero backend — all interactivity is client-side mock data
