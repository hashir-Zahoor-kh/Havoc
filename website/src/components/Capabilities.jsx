import { useState } from 'react'
import { motion } from 'framer-motion'
import { chaosActions, techStack } from '../data/mockData'

const DEPLOYMENT_TARGETS = [
  {
    name: 'LOCAL (kind)',
    tag: 'DEVELOPMENT',
    detail: 'Multi-node cluster via kind. Strimzi Kafka, PostgreSQL, and Redis deploy through Helm in under five minutes. Full experiment pipeline runs with zero cloud spend.',
    pills: ['kind', 'Docker', 'Helm', 'Strimzi'],
  },
  {
    name: 'AWS EKS',
    tag: 'PRODUCTION',
    detail: 'Terraform-managed EKS cluster with Strimzi Kafka operator, RDS PostgreSQL, and ElastiCache Redis. ALB ingress, IAM roles, and VPC networking provisioned automatically.',
    pills: ['EKS', 'Terraform', 'RDS', 'ElastiCache', 'ALB'],
  },
]

const ACTION_ACCENT = {
  POD_KILL:        '#ff3333',
  CPU_PRESSURE:    '#caff00',
  NETWORK_LATENCY: '#caff00',
}

export default function Capabilities() {
  return (
    <section id="capabilities" style={{ background: '#000000', padding: '7rem 2rem', borderBottom: '1px solid #0f0f0f' }}>
      <div style={{ maxWidth: '1200px', margin: '0 auto' }}>

        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }} whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut' }}
          style={{ marginBottom: '3.5rem' }}
        >
          <p style={eyebrow}>// WHAT IT CAN DO</p>
          <h2 style={heading}>CAPABILITIES</h2>
        </motion.div>

        {/* Chaos action cards */}
        <motion.div
          initial={{ opacity: 0, y: 16 }} whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut', delay: 0.05 }}
          style={{ marginBottom: '0.75rem' }}
        >
          <p style={sectionLabel}>CHAOS ACTIONS</p>
        </motion.div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '1px', marginBottom: '5rem', border: '1px solid #1a1a1a' }}>
          {chaosActions.map((action, i) => (
            <ActionCard key={action.name} action={action} delay={i * 0.07} />
          ))}
        </div>

        {/* Tech stack */}
        <motion.div
          initial={{ opacity: 0, y: 16 }} whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut' }}
          style={{ marginBottom: '5rem' }}
        >
          <p style={{ ...sectionLabel, marginBottom: '1.75rem' }}>TECH STACK</p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0' }}>
            {Object.entries(techStack).map(([category, items], i) => (
              <motion.div
                key={category}
                initial={{ opacity: 0, x: -12 }} whileInView={{ opacity: 1, x: 0 }}
                viewport={{ once: true }} transition={{ duration: 0.25, ease: 'easeOut', delay: i * 0.06 }}
                style={{
                  display: 'flex', alignItems: 'stretch',
                  borderTop: '1px solid #111',
                  borderLeft: '1px solid #111',
                  borderRight: '1px solid #111',
                  borderBottom: i === Object.keys(techStack).length - 1 ? '1px solid #111' : 'none',
                }}
              >
                <div style={{
                  width: '140px', flexShrink: 0,
                  borderRight: '1px solid #111',
                  padding: '0.75rem 1rem',
                  display: 'flex', alignItems: 'center',
                }}>
                  <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.71rem', letterSpacing: '0.13em', color: '#444', textTransform: 'uppercase' }}>
                    {category}
                  </span>
                </div>
                <div style={{ padding: '0.65rem 1rem', display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'center' }}>
                  {items.map(item => (
                    <StackPill key={item} label={item} />
                  ))}
                </div>
              </motion.div>
            ))}
          </div>
        </motion.div>

        {/* Deployment targets */}
        <motion.div
          initial={{ opacity: 0, y: 16 }} whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut' }}
        >
          <p style={{ ...sectionLabel, marginBottom: '1.75rem' }}>DEPLOYMENT TARGETS</p>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '1.5rem' }}>
            {DEPLOYMENT_TARGETS.map((target, i) => (
              <DeployCard key={target.name} target={target} delay={i * 0.08} />
            ))}
          </div>
        </motion.div>

      </div>
    </section>
  )
}

function ActionCard({ action, delay }) {
  const [hovered, setHovered] = useState(false)
  const accent = ACTION_ACCENT[action.name] || '#caff00'

  return (
    <motion.div
      initial={{ opacity: 0, y: 14 }} whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }} transition={{ duration: 0.28, ease: 'easeOut', delay }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: '#0a0a0a',
        padding: '2rem 1.75rem',
        position: 'relative',
        cursor: 'default',
        outline: hovered ? `1px solid ${accent}` : '1px solid transparent',
        transition: 'outline-color 150ms ease-out',
      }}
    >
      {/* Tag */}
      <span style={{
        position: 'absolute', top: '1.1rem', right: '1.1rem',
        fontFamily: '"Courier New",monospace', fontSize: '0.65rem',
        letterSpacing: '0.1em', color: hovered ? accent : '#444',
        border: `1px solid ${hovered ? accent : '#222'}`,
        padding: '0.15rem 0.5rem', textTransform: 'uppercase',
        transition: 'all 150ms ease-out',
      }}>
        {action.tag}
      </span>

      {/* Action name */}
      <h3 style={{
        fontFamily: '"Courier New",monospace', fontSize: '1rem',
        letterSpacing: '0.12em', color: hovered ? accent : '#ffffff',
        textTransform: 'uppercase', margin: '0 0 1rem',
        paddingRight: '6rem', fontWeight: 'normal',
        transition: 'color 150ms ease-out',
      }}>
        {action.name}
      </h3>

      {/* Divider */}
      <div style={{ width: '2rem', height: '1px', background: hovered ? accent : '#222', marginBottom: '1rem', transition: 'background 150ms, width 150ms' }} />

      {/* Description */}
      <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.99rem', lineHeight: 1.82, color: '#999', margin: 0 }}>
        {action.description}
      </p>
    </motion.div>
  )
}

function DeployCard({ target, delay }) {
  const [hovered, setHovered] = useState(false)
  const isProd = target.tag === 'PRODUCTION'

  return (
    <motion.div
      initial={{ opacity: 0, y: 14 }} whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }} transition={{ duration: 0.28, ease: 'easeOut', delay }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        border: `1px solid ${hovered ? '#caff00' : '#1a1a1a'}`,
        background: '#0a0a0a', padding: '1.75rem',
        position: 'relative', transition: 'border-color 150ms ease-out',
      }}
    >
      {/* Tag */}
      <span style={{
        position: 'absolute', top: '1rem', right: '1rem',
        fontFamily: '"Courier New",monospace', fontSize: '0.65rem',
        letterSpacing: '0.1em',
        color: isProd ? '#caff00' : '#777',
        border: `1px solid ${isProd ? '#2a3300' : '#1a1a1a'}`,
        padding: '0.15rem 0.5rem', textTransform: 'uppercase',
      }}>
        {target.tag}
      </span>

      <h3 style={{
        fontFamily: '"Courier New",monospace', fontSize: '0.97rem',
        letterSpacing: '0.12em', color: '#ffffff', textTransform: 'uppercase',
        margin: '0 0 0.9rem', paddingRight: '7rem', fontWeight: 'normal',
      }}>
        {target.name}
      </h3>

      <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.99rem', lineHeight: 1.82, color: '#999', margin: '0 0 1.25rem' }}>
        {target.detail}
      </p>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.4rem' }}>
        {target.pills.map(p => <StackPill key={p} label={p} />)}
      </div>
    </motion.div>
  )
}

function StackPill({ label }) {
  const [hovered, setHovered] = useState(false)
  return (
    <span
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        fontFamily: '"Courier New",monospace', fontSize: '0.73rem',
        letterSpacing: '0.08em', textTransform: 'uppercase',
        color: hovered ? '#caff00' : '#777',
        border: `1px solid ${hovered ? '#caff00' : '#222'}`,
        padding: '0.25rem 0.65rem',
        transition: 'color 120ms ease-out, border-color 120ms ease-out',
        cursor: 'default',
      }}
    >
      {label}
    </span>
  )
}

const eyebrow    = { fontFamily: '"Courier New",Courier,monospace', fontSize: '0.83rem', letterSpacing: '0.18em', color: '#caff00', textTransform: 'uppercase', margin: '0 0 0.6rem' }
const heading    = { fontFamily: '"Bebas Neue",cursive', fontSize: 'clamp(3rem,8vw,5.5rem)', color: '#ffffff', margin: 0, letterSpacing: '0.04em', lineHeight: 1 }
const sectionLabel = { fontFamily: '"Courier New",monospace', fontSize: '0.71rem', letterSpacing: '0.18em', color: '#333', textTransform: 'uppercase', margin: '0 0 1.25rem' }
