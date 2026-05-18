import { motion } from 'framer-motion'
import NodeGrid from './NodeGrid'

const fadeUp = (delay = 0) => ({
  initial: { opacity: 0, y: 28 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.35, ease: 'easeOut', delay },
})

export default function Hero() {
  const scrollToDemo = (e) => {
    e.preventDefault()
    document.querySelector('#demo')?.scrollIntoView({ behavior: 'smooth' })
  }

  return (
    <section
      id="overview"
      style={{
        position: 'relative',
        minHeight: '100vh',
        background: '#000000',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        overflow: 'hidden',
        padding: '0 2rem',
      }}
    >
      {/* Animated background */}
      <NodeGrid />

      {/* Vignette so edges fade to black */}
      <div style={{
        position: 'absolute',
        inset: 0,
        background: 'radial-gradient(ellipse 70% 70% at 50% 50%, transparent 40%, #000000 100%)',
        pointerEvents: 'none',
        zIndex: 1,
      }} />

      {/* Content */}
      <div style={{
        position: 'relative',
        zIndex: 2,
        textAlign: 'center',
        maxWidth: '1100px',
        width: '100%',
      }}>
        {/* Eyebrow label */}
        <motion.p {...fadeUp(0)} style={{
          fontFamily: '"Courier New", Courier, monospace',
          fontSize: '0.83rem',
          letterSpacing: '0.2em',
          textTransform: 'uppercase',
          color: '#caff00',
          margin: '0 0 1.2rem',
        }}>
          // CHAOS ENGINEERING PLATFORM
        </motion.p>

        {/* Headline */}
        <motion.h1 {...fadeUp(0.08)} style={{
          fontFamily: '"Bebas Neue", cursive',
          fontSize: 'clamp(5.5rem, 18vw, 16rem)',
          color: '#ffffff',
          margin: '0 0 2rem',
          letterSpacing: '0.04em',
          lineHeight: 0.92,
        }}>
          HAVOC
        </motion.h1>

        {/* Subtext */}
        <motion.div {...fadeUp(0.18)} style={{
          maxWidth: '640px',
          margin: '0 auto 3rem',
        }}>
          <p style={{
            fontFamily: '"Courier New", Courier, monospace',
            fontSize: '1.09rem',
            lineHeight: 1.85,
            color: '#999999',
            margin: 0,
          }}>
            A production-depth Kubernetes chaos engineering platform that injects
            controlled failures — pod kills, CPU pressure, and network latency — across
            your cluster via a Kafka-backed command pipeline.
          </p>
          <p style={{
            fontFamily: '"Courier New", Courier, monospace',
            fontSize: '1.09rem',
            lineHeight: 1.85,
            color: '#999999',
            margin: '0.6rem 0 0',
          }}>
            Four safety guardrails enforce blast radius limits, distributed locks,
            a global kill switch, and scheduled blackout windows before any experiment runs.
          </p>
        </motion.div>

        {/* CTAs */}
        <motion.div {...fadeUp(0.28)} style={{
          display: 'flex',
          gap: '1rem',
          justifyContent: 'center',
          flexWrap: 'wrap',
        }}>
          <RunBtn onClick={scrollToDemo} />
          <GithubBtn />
        </motion.div>

        {/* Scroll hint */}
        <motion.div {...fadeUp(0.42)} style={{
          marginTop: '5rem',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: '0.5rem',
          opacity: 0.35,
        }}>
          <span style={{
            fontFamily: '"Courier New", Courier, monospace',
            fontSize: '0.71rem',
            letterSpacing: '0.18em',
            textTransform: 'uppercase',
            color: '#ffffff',
          }}>SCROLL</span>
          <svg width="12" height="18" viewBox="0 0 12 18" fill="none">
            <line x1="6" y1="0" x2="6" y2="14" stroke="white" strokeWidth="1"/>
            <polyline points="2,10 6,15 10,10" stroke="white" strokeWidth="1" fill="none"/>
          </svg>
        </motion.div>
      </div>
    </section>
  )
}

function RunBtn({ onClick }) {
  return (
    <a
      href="#demo"
      onClick={onClick}
      style={{
        fontFamily: '"Courier New", Courier, monospace',
        fontSize: '0.89rem',
        letterSpacing: '0.13em',
        textTransform: 'uppercase',
        background: '#caff00',
        color: '#000000',
        padding: '0.85rem 2.2rem',
        textDecoration: 'none',
        fontWeight: 700,
        border: '1px solid #caff00',
        transition: 'background 150ms ease-out, border-color 150ms ease-out',
        display: 'inline-block',
      }}
      onMouseEnter={e => {
        e.currentTarget.style.background = '#b8e800'
        e.currentTarget.style.borderColor = '#b8e800'
      }}
      onMouseLeave={e => {
        e.currentTarget.style.background = '#caff00'
        e.currentTarget.style.borderColor = '#caff00'
      }}
    >
      RUN EXPERIMENT
    </a>
  )
}

function GithubBtn() {
  return (
    <a
      href="https://github.com/hashir-zahoor-kh"
      target="_blank"
      rel="noopener noreferrer"
      style={{
        fontFamily: '"Courier New", Courier, monospace',
        fontSize: '0.89rem',
        letterSpacing: '0.13em',
        textTransform: 'uppercase',
        background: 'transparent',
        color: '#ffffff',
        padding: '0.85rem 2.2rem',
        textDecoration: 'none',
        border: '1px solid #ffffff',
        transition: 'color 150ms ease-out, border-color 150ms ease-out',
        display: 'inline-block',
      }}
      onMouseEnter={e => {
        e.currentTarget.style.color = '#caff00'
        e.currentTarget.style.borderColor = '#caff00'
      }}
      onMouseLeave={e => {
        e.currentTarget.style.color = '#ffffff'
        e.currentTarget.style.borderColor = '#ffffff'
      }}
    >
      VIEW ON GITHUB
    </a>
  )
}
