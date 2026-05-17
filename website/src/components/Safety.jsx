import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { blackoutWindows } from '../data/mockData'

const BLAST_LIMIT = 35
const DAYS = ['MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT', 'SUN']
const TOTAL_PODS = 12

const blockedMap = {}
DAYS.forEach(d => { blockedMap[d] = new Set() })
blackoutWindows.forEach(({ day, start, end }) => {
  for (let h = start; h < Math.min(end, 24); h++) blockedMap[day].add(h)
})

const MOCK_EXPERIMENTS = [
  { id: 'hvc-a3f2c1d4', action: 'POD_KILL',        target: 'app=checkout' },
  { id: 'hvc-b8e7d2a1', action: 'CPU_PRESSURE',     target: 'app=payment'  },
  { id: 'hvc-c1d9f4b2', action: 'NETWORK_LATENCY',  target: 'app=gateway'  },
]

export default function Safety() {
  return (
    <section id="safety" style={{ background: '#000000', padding: '7rem 2rem', borderBottom: '1px solid #0f0f0f' }}>
      <div style={{ maxWidth: '1200px', margin: '0 auto' }}>

        <motion.div
          initial={{ opacity: 0, y: 20 }} whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut' }}
          style={{ marginBottom: '3.5rem' }}
        >
          <p style={eyebrow}>// SAFETY FIRST</p>
          <h2 style={heading}>GUARDRAILS</h2>
        </motion.div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(480px, 1fr))', gap: '1.5rem' }}>
          <BlastRadiusCard />
          <LockCard />
          <KillSwitchCard />
          <BlackoutCard />
        </div>

      </div>
    </section>
  )
}

/* ── 1. BLAST RADIUS LIMIT ─────────────────────────────────────────── */
function BlastRadiusCard() {
  const [radius, setRadius] = useState(15)
  const affected  = Math.round(TOTAL_PODS * radius / 100)
  const rejected  = radius > BLAST_LIMIT

  return (
    <GuardrailCard title="BLAST RADIUS LIMIT" tag="PROTECTION">
      <p style={desc}>
        Caps the percentage of matching pods that can be targeted in a single experiment.
        Experiments that would exceed the configured limit are rejected before dispatch.
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: '0.45rem', margin: '1.5rem 0 1rem' }}>
        {Array.from({ length: TOTAL_PODS }, (_, i) => (
          <div
            key={i}
            style={{
              aspectRatio: '1',
              border: `1px solid ${i < affected ? '#ff3333' : '#1a1a1a'}`,
              background:  i < affected ? 'rgba(255,51,51,0.1)' : '#0a0a0a',
              transition: 'border-color 150ms ease-out, background 150ms ease-out',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >
            {i < affected && (
              <span style={{ width: '5px', height: '5px', background: '#ff3333', borderRadius: '50%', display: 'inline-block' }} />
            )}
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.6rem', fontFamily: '"Courier New",monospace', fontSize: '0.63rem', letterSpacing: '0.1em' }}>
        <span style={{ color: '#555' }}>{affected} / {TOTAL_PODS} pods affected</span>
        <span style={{ color: rejected ? '#ff3333' : '#caff00' }}>{radius}%</span>
      </div>

      <input
        type="range" min="0" max="50" value={radius}
        onChange={e => setRadius(+e.target.value)}
        style={{ width: '100%', accentColor: rejected ? '#ff3333' : '#caff00', cursor: 'pointer' }}
      />

      <div style={{ marginTop: '0.75rem', minHeight: '1.4rem' }}>
        <AnimatePresence>
          {rejected && (
            <motion.p
              key="warn"
              initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              style={{ fontFamily: '"Courier New",monospace', fontSize: '0.68rem', color: '#ff3333', margin: 0, letterSpacing: '0.1em' }}
            >
              EXPERIMENT WOULD BE REJECTED — EXCEEDS {BLAST_LIMIT}% LIMIT
            </motion.p>
          )}
        </AnimatePresence>
      </div>
    </GuardrailCard>
  )
}

/* ── 2. ACTIVE EXPERIMENT LOCK ────────────────────────────────────── */
function LockCard() {
  const [locked, setLocked] = useState(true)

  useEffect(() => {
    let timer
    const cycle = () => {
      setLocked(true)
      timer = setTimeout(() => {
        setLocked(false)
        timer = setTimeout(cycle, 1800)
      }, 3000)
    }
    cycle()
    return () => clearTimeout(timer)
  }, [])

  return (
    <GuardrailCard title="ACTIVE EXPERIMENT LOCK" tag="REDIS">
      <p style={desc}>
        A Redis key with a TTL is acquired before any experiment runs. While the lock is held,
        concurrent experiments targeting the same service are blocked, preventing compound failures.
      </p>

      <div style={{ margin: '1.5rem 0 1rem', background: '#050505', border: '1px solid #111', padding: '1rem 1.25rem', fontFamily: '"Courier New",monospace', fontSize: '0.7rem', minHeight: '96px' }}>
        <span style={{ color: '#555', display: 'block', marginBottom: '0.6rem', fontSize: '0.55rem', letterSpacing: '0.1em' }}>redis-cli</span>

        <AnimatePresence mode="wait">
          {locked ? (
            <motion.div key="locked"
              initial={{ opacity: 0, x: -8 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: 8 }}
              transition={{ duration: 0.25 }}
            >
              <span style={{ color: '#caff00' }}>SET </span>
              <span style={{ color: '#fff' }}>havoc:active:checkout </span>
              <span style={{ color: '#888' }}>1 EX 300</span>
              <div style={{ marginTop: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                <span className="pulse-dot" style={{ width: '6px', height: '6px', background: '#caff00', borderRadius: '50%', display: 'inline-block', flexShrink: 0 }} />
                <span style={{ color: '#caff00', letterSpacing: '0.1em', fontSize: '0.65rem' }}>LOCK ACQUIRED</span>
              </div>
            </motion.div>
          ) : (
            <motion.div key="unlocked"
              initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.25 }}
            >
              <span style={{ color: '#555' }}>DEL havoc:active:checkout</span>
              <div style={{ marginTop: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                <span style={{ width: '6px', height: '6px', background: '#333', borderRadius: '50%', display: 'inline-block' }} />
                <span style={{ color: '#444', letterSpacing: '0.1em', fontSize: '0.65rem' }}>LOCK RELEASED</span>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.6rem', color: '#333', letterSpacing: '0.08em', margin: 0 }}>
        TTL: 300s · renewed on heartbeat · auto-expires on agent crash
      </p>
    </GuardrailCard>
  )
}

/* ── 3. GLOBAL KILL SWITCH ────────────────────────────────────────── */
function KillSwitchCard() {
  const fresh = () => MOCK_EXPERIMENTS.map(e => ({ ...e, status: 'RUNNING' }))
  const [experiments, setExperiments] = useState(fresh)
  const [killed, setKilled] = useState(false)

  const handleKill = () => {
    setKilled(true)
    setExperiments(prev => prev.map(e => ({ ...e, status: 'ABORTING' })))
    setTimeout(() => {
      setExperiments(prev => prev.map(e => ({ ...e, status: 'ABORTED' })))
    }, 700)
  }

  const handleReset = () => {
    setKilled(false)
    setExperiments(fresh())
  }

  return (
    <GuardrailCard title="GLOBAL KILL SWITCH" tag="EMERGENCY">
      <p style={desc}>
        A Redis flag that every agent reads before executing each action step. Setting it halts
        all in-progress experiments within one polling cycle (&lt;500ms) across the entire cluster.
      </p>

      <div style={{ margin: '1.5rem 0 1.25rem', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
        {experiments.map(exp => {
          const color = { RUNNING: '#caff00', ABORTING: '#ff9900', ABORTED: '#ff3333' }[exp.status]
          return (
            <motion.div
              key={exp.id}
              animate={{ borderColor: exp.status === 'ABORTED' ? '#ff3333' : exp.status === 'ABORTING' ? '#ff9900' : '#1a1a1a' }}
              transition={{ duration: 0.3 }}
              style={{ border: '1px solid #1a1a1a', padding: '0.55rem 0.85rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontFamily: '"Courier New",monospace', fontSize: '0.65rem' }}
            >
              <div>
                <span style={{ color: '#fff', letterSpacing: '0.08em' }}>{exp.action}</span>
                <span style={{ color: '#444', margin: '0 0.5rem' }}>—</span>
                <span style={{ color: '#666' }}>{exp.target}</span>
              </div>
              <span style={{ color, letterSpacing: '0.1em', display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                {exp.status === 'RUNNING' && (
                  <span className="pulse-dot" style={{ width: '5px', height: '5px', background: '#caff00', borderRadius: '50%', display: 'inline-block' }} />
                )}
                {exp.status}
              </span>
            </motion.div>
          )
        })}
      </div>

      <div style={{ display: 'flex', gap: '0.75rem' }}>
        <button
          onClick={killed ? handleReset : handleKill}
          style={{
            flex: 1,
            fontFamily: '"Courier New",monospace', fontSize: '0.7rem', letterSpacing: '0.12em',
            textTransform: 'uppercase', padding: '0.75rem',
            border: `1px solid ${killed ? '#333' : '#ff3333'}`,
            color: killed ? '#555' : '#ff3333',
            background: 'transparent', cursor: 'pointer',
            transition: 'all 150ms ease-out',
          }}
          onMouseEnter={e => {
            if (!killed) { e.currentTarget.style.background = 'rgba(255,51,51,0.08)' }
          }}
          onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
        >
          {killed ? 'RESET EXPERIMENTS' : 'STOP ALL EXPERIMENTS'}
        </button>
      </div>
    </GuardrailCard>
  )
}

/* ── 4. BLACKOUT WINDOWS ──────────────────────────────────────────── */
function BlackoutCard() {
  const [hovered, setHovered] = useState(null)

  return (
    <GuardrailCard title="BLACKOUT WINDOWS" tag="SCHEDULING">
      <p style={desc}>
        A weekly schedule of protected time windows. Any experiment that would start during a
        blackout period is rejected by the control plane before acquiring a lock or publishing commands.
      </p>

      <div style={{ margin: '1.5rem 0 0.5rem', overflowX: 'auto' }}>
        {/* Hour axis labels */}
        <div style={{ display: 'flex', marginBottom: '4px', paddingLeft: '34px', gap: '0' }}>
          {[0, 6, 12, 18].map(h => (
            <div key={h} style={{ flex: '6 0 0', fontFamily: '"Courier New",monospace', fontSize: '0.48rem', color: '#333', letterSpacing: '0.06em' }}>
              {String(h).padStart(2, '0')}:00
            </div>
          ))}
        </div>

        {DAYS.map(day => (
          <div key={day} style={{ display: 'flex', alignItems: 'center', gap: '4px', marginBottom: '3px' }}>
            <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.52rem', color: '#444', letterSpacing: '0.06em', width: '30px', flexShrink: 0 }}>
              {day}
            </span>
            {Array.from({ length: 24 }, (_, h) => {
              const blocked = blockedMap[day].has(h)
              const isHov   = hovered?.day === day && hovered?.hour === h
              return (
                <div
                  key={h}
                  onMouseEnter={() => blocked && setHovered({ day, hour: h })}
                  onMouseLeave={() => setHovered(null)}
                  style={{
                    flex: 1,
                    height: '16px',
                    background: blocked ? (isHov ? 'rgba(255,51,51,0.25)' : 'rgba(255,51,51,0.12)') : '#0a0a0a',
                    border:     `1px solid ${blocked ? '#3a0000' : '#111'}`,
                    cursor:     blocked ? 'crosshair' : 'default',
                    transition: 'background 120ms',
                  }}
                />
              )
            })}
          </div>
        ))}

        {/* Legend */}
        <div style={{ display: 'flex', gap: '1.25rem', marginTop: '0.75rem' }}>
          {[['#3a0000', 'rgba(255,51,51,0.12)', 'BLOCKED'], ['#111', '#0a0a0a', 'AVAILABLE']].map(([border, bg, label]) => (
            <div key={label} style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
              <div style={{ width: '12px', height: '12px', border: `1px solid ${border}`, background: bg }} />
              <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.55rem', color: '#444', letterSpacing: '0.08em' }}>{label}</span>
            </div>
          ))}
        </div>
      </div>

      <div style={{ minHeight: '1.4rem', marginTop: '0.25rem' }}>
        <AnimatePresence>
          {hovered && (
            <motion.p
              key="tooltip"
              initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.15 }}
              style={{ fontFamily: '"Courier New",monospace', fontSize: '0.65rem', color: '#ff3333', margin: 0, letterSpacing: '0.08em' }}
            >
              EXPERIMENT REJECTED — BLACKOUT ACTIVE ({hovered.day} {String(hovered.hour).padStart(2,'0')}:00)
            </motion.p>
          )}
        </AnimatePresence>
      </div>
    </GuardrailCard>
  )
}

/* ── Shared components ──────────────────────────────────────────────── */
function GuardrailCard({ title, tag, children }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 16 }} whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut' }}
      style={{ border: '1px solid #1a1a1a', background: '#0a0a0a', padding: '1.75rem', position: 'relative' }}
    >
      <span style={{
        position: 'absolute', top: '1rem', right: '1rem',
        fontFamily: '"Courier New",monospace', fontSize: '0.55rem',
        letterSpacing: '0.1em', color: '#444', border: '1px solid #1a1a1a',
        padding: '0.15rem 0.5rem', textTransform: 'uppercase',
      }}>{tag}</span>

      <h3 style={{
        fontFamily: '"Courier New",monospace', fontSize: '0.78rem',
        letterSpacing: '0.12em', color: '#ffffff', textTransform: 'uppercase',
        margin: '0 0 0.75rem', paddingRight: '5rem', fontWeight: 'normal',
      }}>{title}</h3>

      {children}
    </motion.div>
  )
}

/* ── Shared styles ──────────────────────────────────────────────────── */
const eyebrow = { fontFamily: '"Courier New",Courier,monospace', fontSize: '0.7rem', letterSpacing: '0.18em', color: '#caff00', textTransform: 'uppercase', margin: '0 0 0.6rem' }
const heading  = { fontFamily: '"Bebas Neue",cursive', fontSize: 'clamp(3rem,8vw,5.5rem)', color: '#ffffff', margin: 0, letterSpacing: '0.04em', lineHeight: 1 }
const desc     = { fontFamily: '"Courier New",monospace', fontSize: '0.72rem', lineHeight: 1.7, color: '#555', margin: '0 0 0.25rem' }
