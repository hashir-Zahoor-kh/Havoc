import { useState } from 'react'
import { motion } from 'framer-motion'

const GITHUB_URL   = 'https://github.com/hashir-zahoor-kh'
const LINKEDIN_URL = 'https://www.linkedin.com/in/hashirzahoor'
const EMAIL        = 'hashirzahoor74@icloud.com'
const NAME         = 'Hashir Zahoor ur Rahman'

export default function About() {
  return (
    <>
      <section id="about" style={{ background: '#000000', padding: '7rem 2rem 0', borderBottom: '1px solid #0f0f0f' }}>
        <div style={{ maxWidth: '1200px', margin: '0 auto' }}>

          {/* Header */}
          <motion.div
            initial={{ opacity: 0, y: 20 }} whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut' }}
            style={{ marginBottom: '4rem' }}
          >
            <p style={eyebrow}>// THE PROJECT</p>
            <h2 style={heading}>ABOUT</h2>
          </motion.div>

          {/* Two-column layout */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '4rem', paddingBottom: '6rem' }}>

            {/* Left — description */}
            <motion.div
              initial={{ opacity: 0, y: 16 }} whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut', delay: 0.05 }}
            >
              <p style={bodyText}>
                Havoc is a portfolio project built to demonstrate production-depth distributed
                systems design. Every architectural decision — from Kafka-backed command streaming
                to DaemonSet-per-node agents — mirrors how real chaos engineering platforms
                are structured in the industry.
              </p>
              <p style={{ ...bodyText, marginTop: '1.25rem' }}>
                The platform covers the full engineering stack: a Go binary pipeline, Kubernetes
                operator patterns, Terraform-managed cloud infrastructure, Helm packaging, and
                a safety-first guardrail system enforced at multiple layers before any failure
                is injected.
              </p>
              <p style={{ ...bodyText, marginTop: '1.25rem' }}>
                The same Go binaries and Helm charts run on kind locally and AWS EKS in
                production — only the environment values files differ between targets, not
                the application code itself.
              </p>

              {/* Stat strip */}
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0', marginTop: '2.5rem', border: '1px solid #1a1a1a' }}>
                {[
                  ['3', 'GO BINARIES'],
                  ['4', 'GUARDRAILS'],
                  ['3', 'CHAOS ACTIONS'],
                  ['2', 'DEPLOY TARGETS'],
                ].map(([num, label], i, arr) => (
                  <div
                    key={label}
                    style={{
                      flex: '1 0 0',
                      minWidth: '100px',
                      padding: '1.25rem 1rem',
                      borderRight: i < arr.length - 1 ? '1px solid #1a1a1a' : 'none',
                      textAlign: 'center',
                    }}
                  >
                    <div style={{ fontFamily: '"Bebas Neue",cursive', fontSize: '2.5rem', color: '#caff00', lineHeight: 1 }}>{num}</div>
                    <div style={{ fontFamily: '"Courier New",monospace', fontSize: '0.6rem', letterSpacing: '0.12em', color: '#666', textTransform: 'uppercase', marginTop: '0.3rem' }}>{label}</div>
                  </div>
                ))}
              </div>
            </motion.div>

            {/* Right — contact card */}
            <motion.div
              initial={{ opacity: 0, y: 16 }} whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut', delay: 0.12 }}
            >
              <div style={{ border: '1px solid #1a1a1a', background: '#0a0a0a', padding: '2.25rem' }}>

                <h3 style={{
                  fontFamily: '"Bebas Neue",cursive',
                  fontSize: 'clamp(1.8rem, 4vw, 2.8rem)',
                  color: '#ffffff', margin: '0 0 0.5rem',
                  letterSpacing: '0.04em', lineHeight: 1.1,
                }}>
                  INTERESTED IN<br />WORKING TOGETHER?
                </h3>

                <p style={{ ...bodyText, marginTop: '1rem', marginBottom: '2rem' }}>
                  I'm a software engineer focused on platform engineering, distributed systems,
                  and cloud-native infrastructure. Open to full-time roles and interesting projects.
                </p>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                  {/* Email */}
                  <ContactRow label="EMAIL">
                    <a
                      href={`mailto:${EMAIL}`}
                      style={contactLink}
                      onMouseEnter={e => { e.currentTarget.style.color = '#caff00' }}
                      onMouseLeave={e => { e.currentTarget.style.color = '#ffffff' }}
                    >
                      {EMAIL}
                    </a>
                  </ContactRow>

                  {/* GitHub */}
                  <ContactRow label="GITHUB">
                    <a
                      href={GITHUB_URL}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={contactLink}
                      onMouseEnter={e => { e.currentTarget.style.color = '#caff00' }}
                      onMouseLeave={e => { e.currentTarget.style.color = '#ffffff' }}
                    >
                      {GITHUB_URL.replace('https://', '')}
                    </a>
                  </ContactRow>

                  {/* LinkedIn */}
                  <ContactRow label="LINKEDIN">
                    <a
                      href={LINKEDIN_URL}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={contactLink}
                      onMouseEnter={e => { e.currentTarget.style.color = '#caff00' }}
                      onMouseLeave={e => { e.currentTarget.style.color = '#ffffff' }}
                    >
                      {LINKEDIN_URL.replace('https://', '')}
                    </a>
                  </ContactRow>
                </div>

                <div style={{ display: 'flex', gap: '0.75rem', marginTop: '2rem', flexWrap: 'wrap' }}>
                  <LinkButton href={`mailto:${EMAIL}`} label="SEND EMAIL" />
                  <LinkButton href={GITHUB_URL} label="VIEW GITHUB" external />
                  <LinkButton href={LINKEDIN_URL} label="LINKEDIN" external />
                </div>

              </div>

              {/* Name card */}
              <div style={{
                borderLeft: '1px solid #1a1a1a',
                borderRight: '1px solid #1a1a1a',
                borderBottom: '1px solid #1a1a1a',
                padding: '1rem 2.25rem',
                display: 'flex', alignItems: 'center', gap: '1rem',
              }}>
                <div style={{ width: '32px', height: '32px', background: '#caff00', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <span style={{ fontFamily: '"Bebas Neue",cursive', fontSize: '1rem', color: '#000', lineHeight: 1 }}>
                    {NAME.split(' ').map(w => w[0]).slice(0,2).join('')}
                  </span>
                </div>
                <div>
                  <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.82rem', color: '#ffffff', margin: 0, letterSpacing: '0.06em' }}>{NAME}</p>
                  <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.65rem', color: '#666', margin: '0.15rem 0 0', letterSpacing: '0.08em', textTransform: 'uppercase' }}>Platform Engineer</p>
                </div>
              </div>
            </motion.div>

          </div>
        </div>
      </section>

      {/* Footer */}
      <footer style={{ background: '#000000', borderTop: '1px solid #1a1a1a', padding: '1.5rem 2rem' }}>
        <div style={{
          maxWidth: '1200px', margin: '0 auto',
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          flexWrap: 'wrap', gap: '0.5rem',
        }}>
          <span style={footerText}>HAVOC — CONTROLLED CHAOS FOR KUBERNETES</span>
          <span style={footerText}>BUILT BY {NAME.toUpperCase()}</span>
        </div>
      </footer>
    </>
  )
}

function ContactRow({ label, children }) {
  return (
    <div style={{ display: 'flex', alignItems: 'baseline', gap: '1rem', borderBottom: '1px solid #111', paddingBottom: '0.65rem' }}>
      <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.58rem', letterSpacing: '0.13em', color: '#444', textTransform: 'uppercase', minWidth: '56px', flexShrink: 0 }}>
        {label}
      </span>
      {children}
    </div>
  )
}

function LinkButton({ href, label, external }) {
  const [hovered, setHovered] = useState(false)
  return (
    <a
      href={href}
      target={external ? '_blank' : undefined}
      rel={external ? 'noopener noreferrer' : undefined}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        fontFamily: '"Courier New",monospace', fontSize: '0.68rem',
        letterSpacing: '0.13em', textTransform: 'uppercase',
        color: hovered ? '#000000' : '#ffffff',
        background: hovered ? '#caff00' : 'transparent',
        border: `1px solid ${hovered ? '#caff00' : '#333'}`,
        padding: '0.65rem 1.25rem',
        textDecoration: 'none',
        transition: 'all 150ms ease-out',
        display: 'inline-block',
      }}
    >
      {label}
    </a>
  )
}

const eyebrow    = { fontFamily: '"Courier New",Courier,monospace', fontSize: '0.7rem', letterSpacing: '0.18em', color: '#caff00', textTransform: 'uppercase', margin: '0 0 0.6rem' }
const heading    = { fontFamily: '"Bebas Neue",cursive', fontSize: 'clamp(3rem,8vw,5.5rem)', color: '#ffffff', margin: 0, letterSpacing: '0.04em', lineHeight: 1 }
const bodyText   = { fontFamily: '"Courier New",monospace', fontSize: '0.88rem', lineHeight: 1.85, color: '#999', margin: 0 }
const contactLink = { fontFamily: '"Courier New",monospace', fontSize: '0.82rem', color: '#ffffff', textDecoration: 'none', letterSpacing: '0.04em', transition: 'color 150ms ease-out' }
const footerText  = { fontFamily: '"Courier New",monospace', fontSize: '0.65rem', letterSpacing: '0.12em', color: '#555', textTransform: 'uppercase' }
