import { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { experimentHistory } from '../data/mockData'

const ACTIONS = ['POD_KILL', 'CPU_PRESSURE', 'NETWORK_LATENCY']
const BLAST_LIMIT = 35

const CHECKS = [
  { label: 'checking blast radius...',        tag: '[PASS]', icon: '✓' },
  { label: 'acquiring service lock...',       tag: '[PASS]', icon: '✓' },
  { label: 'checking blackout windows...',    tag: '[PASS]', icon: '✓' },
  { label: 'publishing to havoc.commands...', tag: '[OK]',   icon: ''  },
]

function makeExpId() {
  return 'hvc-' + Math.random().toString(16).slice(2, 10)
}

function generatePods(selector) {
  const app = (selector.split('=')[1] || 'pod').slice(0, 8)
  return Array.from({ length: 8 }, (_, i) => ({
    name: `${app}-${Math.random().toString(16).slice(2, 7)}`,
    status: 'RUNNING',
  }))
}

const STATUS_COLORS = {
  RUNNING:     { border: '#222',    dot: '#00ff88', text: '#666' },
  TERMINATING: { border: '#ff3333', dot: '#ff3333', text: '#ff3333' },
  PENDING:     { border: '#caff00', dot: '#caff00', text: '#caff00' },
}

const EXP_STATE_COLORS = {
  PENDING:   '#caff00',
  RUNNING:   '#caff00',
  COMPLETED: '#00ff88',
}

export default function Demo() {
  const [action, setAction]             = useState('POD_KILL')
  const [labelSelector, setLabelSelector] = useState('app=checkout')
  const [blastRadius, setBlastRadius]   = useState(20)
  const [duration, setDuration]         = useState(30)

  const [runState, setRunState]     = useState('idle')
  const [visibleChecks, setVisible] = useState(0)
  const [expState, setExpState]     = useState(null)
  const [pods, setPods]             = useState(() => generatePods('app=checkout'))
  const [chaosPod, setChaosPod]     = useState(1)
  const [resultCard, setResultCard] = useState(null)
  const [expId, setExpId]           = useState(null)
  const [expandedRow, setExpanded]  = useState(null)

  const timers = useRef([])
  const after = (fn, ms) => { const id = setTimeout(fn, ms); timers.current.push(id) }
  const clearAll = () => { timers.current.forEach(clearTimeout); timers.current = [] }

  useEffect(() => () => clearAll(), [])
  useEffect(() => { setPods(generatePods(labelSelector)) }, [labelSelector])

  const handleRun = () => {
    clearAll()
    const idx   = Math.floor(Math.random() * 4) + 2
    const id    = makeExpId()
    const fresh = generatePods(labelSelector)
    setChaosPod(idx)
    setPods(fresh)
    setExpId(id)
    setRunState('checking')
    setVisible(0)
    setExpState(null)
    setResultCard(null)

    const rejected = blastRadius > BLAST_LIMIT

    after(() => {
      setVisible(1)
      if (rejected) { setRunState('rejected'); return }

      after(() => { setVisible(2)
        after(() => { setVisible(3)
          after(() => { setVisible(4)
            setRunState('running')
            setExpState('PENDING')

            after(() => {
              setExpState('RUNNING')
              setPods(prev => prev.map((p, i) => i === idx ? { ...p, status: 'TERMINATING' } : p))

              after(() => {
                setPods(prev => prev.map((p, i) => i === idx ? { ...p, status: 'PENDING' } : p))

                after(() => {
                  setPods(prev => prev.map((p, i) => i === idx ? { ...p, status: 'RUNNING' } : p))
                  setExpState('COMPLETED')
                  setRunState('completed')
                  const dur = action === 'POD_KILL' ? '2.3s' : `${duration}.0s`
                  setResultCard({ id, action, target: labelSelector, affected: '1 / 4', status: 'COMPLETED', duration: dur })
                }, 900)
              }, 900)
            }, 600)
          }, 600)
        }, 600)
      }, 600)
    }, 600)
  }

  const showDuration = action !== 'POD_KILL'

  return (
    <section id="demo" style={{ background: '#000000', padding: '7rem 2rem', borderBottom: '1px solid #0f0f0f' }}>
      <div style={{ maxWidth: '1200px', margin: '0 auto' }}>

        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }} whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }} transition={{ duration: 0.3, ease: 'easeOut' }}
          style={{ marginBottom: '3.5rem' }}
        >
          <p style={eyebrow}>// EXPERIMENT CONTROL PANEL</p>
          <h2 style={heading}>DEMO</h2>
        </motion.div>

        {/* Two-column panel */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(340px, 1fr))', gap: '1.5rem', marginBottom: '3rem' }}>

          {/* LEFT — Configure */}
          <div style={panel}>
            <p style={panelLabel}>// CONFIGURE EXPERIMENT</p>

            <Field label="CHAOS ACTION">
              <ActionSelect value={action} onChange={setAction} />
            </Field>

            <Field label="LABEL SELECTOR">
              <input
                value={labelSelector}
                onChange={e => setLabelSelector(e.target.value)}
                placeholder="app=checkout"
                style={inputStyle}
              />
            </Field>

            <Field label={`BLAST RADIUS LIMIT — `} accent={`${blastRadius}%`}>
              <input
                type="range" min="10" max="50" value={blastRadius}
                onChange={e => setBlastRadius(+e.target.value)}
                style={{ width: '100%', accentColor: blastRadius > BLAST_LIMIT ? '#ff3333' : '#caff00', cursor: 'pointer' }}
              />
              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.3rem' }}>
                {[10, 20, 30, 40, 50].map(v => (
                  <span key={v} style={{ fontFamily: 'mono', fontSize: '0.55rem', color: v <= blastRadius ? (blastRadius > BLAST_LIMIT ? '#ff3333' : '#caff00') : '#333', fontFamily: '"Courier New",monospace', letterSpacing: '0.06em' }}>{v}%</span>
                ))}
              </div>
              {blastRadius > BLAST_LIMIT && (
                <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.6rem', color: '#ff3333', margin: '0.4rem 0 0', letterSpacing: '0.08em' }}>
                  WARNING: exceeds system limit of {BLAST_LIMIT}%
                </p>
              )}
            </Field>

            {showDuration && (
              <Field label="DURATION (SECONDS)">
                <input
                  type="number" min="5" max="300" value={duration}
                  onChange={e => setDuration(+e.target.value)}
                  style={inputStyle}
                />
              </Field>
            )}

            <button
              onClick={handleRun}
              style={{
                width: '100%', marginTop: '1.5rem',
                background: '#caff00', color: '#000000',
                border: '1px solid #caff00',
                fontFamily: '"Courier New",Courier,monospace',
                fontSize: '0.75rem', letterSpacing: '0.14em',
                textTransform: 'uppercase', padding: '0.9rem',
                cursor: 'pointer', fontWeight: 700,
                transition: 'background 120ms ease-out',
              }}
              onMouseEnter={e => { e.currentTarget.style.background = '#b8e800' }}
              onMouseLeave={e => { e.currentTarget.style.background = '#caff00' }}
            >
              RUN EXPERIMENT
            </button>
          </div>

          {/* RIGHT — Output */}
          <div style={panel}>
            <p style={panelLabel}>// EXPERIMENT OUTPUT</p>

            {runState === 'idle' && (
              <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.8rem', color: '#888', letterSpacing: '0.1em' }}>
                AWAITING INPUT...
              </p>
            )}

            {/* Guardrail checks */}
            {(runState === 'checking' || runState === 'rejected' || runState === 'running' || runState === 'completed') && (
              <div style={{ marginBottom: '1.5rem', fontFamily: '"Courier New",monospace', fontSize: '0.82rem' }}>
                {CHECKS.slice(0, visibleChecks).map((c, i) => {
                  const isReject = i === 0 && runState === 'rejected'
                  return (
                    <motion.div
                      key={i}
                      initial={{ opacity: 0, x: -8 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ duration: 0.2, ease: 'easeOut' }}
                      style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem', gap: '0.5rem' }}
                    >
                      <span style={{ color: '#999', whiteSpace: 'nowrap' }}>{'>'} {c.label}</span>
                      <span style={{ color: isReject ? '#ff3333' : '#00ff88', whiteSpace: 'nowrap', flexShrink: 0 }}>
                        {isReject ? '[REJECTED]' : `${c.tag} ${c.icon}`}
                      </span>
                    </motion.div>
                  )
                })}
                {runState === 'rejected' && (
                  <motion.p
                    initial={{ opacity: 0 }} animate={{ opacity: 1 }}
                    transition={{ duration: 0.25, delay: 0.1 }}
                    style={{ color: '#ff3333', margin: '0.6rem 0 0', fontSize: '0.68rem', lineHeight: 1.6, letterSpacing: '0.06em' }}
                  >
                    ERROR: blast radius {blastRadius}% exceeds limit of {BLAST_LIMIT}%.<br />experiment aborted.
                  </motion.p>
                )}
              </div>
            )}

            {/* Experiment state badge */}
            {expState && (
              <motion.div
                initial={{ opacity: 0 }} animate={{ opacity: 1 }}
                transition={{ duration: 0.2 }}
                style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1.5rem' }}
              >
                <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.6rem', color: '#666', letterSpacing: '0.12em' }}>STATUS</span>
                <span style={{
                  fontFamily: '"Courier New",monospace', fontSize: '0.78rem', letterSpacing: '0.14em',
                  color: EXP_STATE_COLORS[expState],
                  border: `1px solid ${EXP_STATE_COLORS[expState]}`,
                  padding: '0.2rem 0.7rem',
                }}>
                  {expState}
                </span>
              </motion.div>
            )}

            {/* Pod grid */}
            {(runState === 'running' || runState === 'completed') && (
              <motion.div
                initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25 }}
                style={{ marginBottom: '1.5rem' }}
              >
                <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.6rem', color: '#666', letterSpacing: '0.12em', margin: '0 0 0.75rem', textTransform: 'uppercase' }}>
                  POD GRID — {labelSelector}
                </p>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '0.5rem' }}>
                  {pods.map((pod, i) => {
                    const colors = STATUS_COLORS[pod.status]
                    return (
                      <div key={i} style={{
                        border: `1px solid ${colors.border}`,
                        background: '#0a0a0a',
                        padding: '0.5rem 0.4rem',
                        transition: 'border-color 300ms ease-out',
                      }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.3rem', marginBottom: '0.25rem' }}>
                          <span style={{ width: '5px', height: '5px', background: colors.dot, borderRadius: '50%', flexShrink: 0, display: 'inline-block', transition: 'background 300ms' }} />
                          <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.5rem', color: '#888', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {pod.name}
                          </span>
                        </div>
                        <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.52rem', color: colors.text, letterSpacing: '0.06em', transition: 'color 300ms' }}>
                          {pod.status}
                        </span>
                      </div>
                    )
                  })}
                </div>
              </motion.div>
            )}

            {/* Result card */}
            {resultCard && (
              <motion.div
                initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25 }}
                style={{ border: '1px solid #222', background: '#0a0a0a', padding: '1rem 1.25rem' }}
              >
                {[
                  ['EXPERIMENT_ID', resultCard.id],
                  ['ACTION',        resultCard.action],
                  ['TARGET',        resultCard.target],
                  ['AFFECTED PODS', resultCard.affected],
                  ['STATUS',        resultCard.status],
                  ['DURATION',      resultCard.duration],
                ].map(([k, v]) => (
                  <div key={k} style={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: '0.5rem', marginBottom: '0.35rem' }}>
                    <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.62rem', color: '#666', letterSpacing: '0.1em', textTransform: 'uppercase' }}>{k}</span>
                    <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.62rem', color: k === 'STATUS' ? '#00ff88' : '#ffffff', letterSpacing: '0.06em' }}>{v}</span>
                  </div>
                ))}
              </motion.div>
            )}
          </div>
        </div>

        {/* Experiment history */}
        <div>
          <p style={{ ...eyebrow, marginBottom: '1rem' }}>// EXPERIMENT HISTORY</p>
          <div style={{ border: '1px solid #1a1a1a', overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontFamily: '"Courier New",monospace', fontSize: '0.65rem' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #1a1a1a' }}>
                  {['ID', 'ACTION', 'TARGET', 'STATUS', 'TIMESTAMP', 'DURATION'].map(h => (
                    <th key={h} style={{ padding: '0.65rem 1rem', textAlign: 'left', letterSpacing: '0.12em', color: '#666', fontWeight: 'normal', whiteSpace: 'nowrap' }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {experimentHistory.map(exp => (
                  <HistoryRow
                    key={exp.id}
                    exp={exp}
                    isExpanded={expandedRow === exp.id}
                    onToggle={() => setExpanded(expandedRow === exp.id ? null : exp.id)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </div>

      </div>
    </section>
  )
}

function HistoryRow({ exp, isExpanded, onToggle }) {
  const statusColor = { COMPLETED: '#00ff88', ABORTED: '#ff3333', RUNNING: '#caff00' }[exp.status] || '#888'
  return (
    <>
      <tr
        onClick={onToggle}
        style={{ borderBottom: '1px solid #0f0f0f', cursor: 'pointer', background: isExpanded ? '#0a0a0a' : 'transparent', transition: 'background 150ms' }}
        onMouseEnter={e => { if (!isExpanded) e.currentTarget.style.background = '#080808' }}
        onMouseLeave={e => { if (!isExpanded) e.currentTarget.style.background = 'transparent' }}
      >
        <td style={td}><span style={{ color: '#888' }}>{exp.id}</span></td>
        <td style={td}><span style={{ color: '#fff', letterSpacing: '0.08em' }}>{exp.action}</span></td>
        <td style={td}><span style={{ color: '#888' }}>{exp.target}</span></td>
        <td style={td}>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: '0.4rem' }}>
            {exp.status === 'RUNNING' && <span className="pulse-dot" style={{ display: 'inline-block', width: '5px', height: '5px', background: '#caff00', borderRadius: '50%', flexShrink: 0 }} />}
            <span style={{ color: statusColor, letterSpacing: '0.08em' }}>{exp.status}</span>
          </span>
        </td>
        <td style={td}><span style={{ color: '#888' }}>{exp.timestamp}</span></td>
        <td style={td}><span style={{ color: '#888' }}>{exp.duration}</span></td>
      </tr>
      {isExpanded && (
        <tr style={{ borderBottom: '1px solid #0f0f0f' }}>
          <td colSpan={6} style={{ padding: '0.75rem 1rem 1rem', background: '#0a0a0a' }}>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '0.5rem' }}>
              {[
                ['EXPERIMENT ID', exp.id],
                ['ACTION',        exp.action],
                ['TARGET',        exp.target],
                ['STATUS',        exp.status],
                ['TIMESTAMP',     exp.timestamp],
                ['DURATION',      exp.duration],
              ].map(([k, v]) => (
                <div key={k}>
                  <span style={{ display: 'block', fontFamily: '"Courier New",monospace', fontSize: '0.55rem', color: '#666', letterSpacing: '0.1em', marginBottom: '0.15rem' }}>{k}</span>
                  <span style={{ fontFamily: '"Courier New",monospace', fontSize: '0.62rem', color: k === 'STATUS' ? ({ COMPLETED: '#00ff88', ABORTED: '#ff3333', RUNNING: '#caff00' }[v] || '#fff') : '#fff' }}>{v}</span>
                </div>
              ))}
            </div>
          </td>
        </tr>
      )}
    </>
  )
}

function ActionSelect({ value, onChange }) {
  const [open, setOpen] = useState(false)
  return (
    <div style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen(o => !o)}
        style={{ ...inputStyle, width: '100%', textAlign: 'left', cursor: 'pointer', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
      >
        <span>{value}</span>
        <span style={{ color: '#666' }}>▾</span>
      </button>
      {open && (
        <div style={{ position: 'absolute', top: '100%', left: 0, right: 0, background: '#0a0a0a', border: '1px solid #333', zIndex: 20 }}>
          {ACTIONS.map(a => (
            <button
              key={a}
              onClick={() => { onChange(a); setOpen(false) }}
              style={{ display: 'block', width: '100%', textAlign: 'left', background: a === value ? '#111' : 'transparent', border: 'none', padding: '0.65rem 0.85rem', fontFamily: '"Courier New",monospace', fontSize: '0.7rem', color: a === value ? '#caff00' : '#fff', cursor: 'pointer', letterSpacing: '0.08em' }}
              onMouseEnter={e => { if (a !== value) e.currentTarget.style.background = '#111' }}
              onMouseLeave={e => { if (a !== value) e.currentTarget.style.background = 'transparent' }}
            >
              {a}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function Field({ label, accent, children }) {
  return (
    <div style={{ marginBottom: '1.25rem' }}>
      <p style={{ fontFamily: '"Courier New",monospace', fontSize: '0.7rem', letterSpacing: '0.13em', color: '#777', textTransform: 'uppercase', margin: '0 0 0.5rem' }}>
        {label}
        {accent && <span style={{ color: '#caff00' }}>{accent}</span>}
      </p>
      {children}
    </div>
  )
}

/* ---- shared styles ---- */
const eyebrow = { fontFamily: '"Courier New",Courier,monospace', fontSize: '0.7rem', letterSpacing: '0.18em', color: '#caff00', textTransform: 'uppercase', margin: '0 0 0.6rem' }
const heading  = { fontFamily: '"Bebas Neue",cursive', fontSize: 'clamp(3rem,8vw,5.5rem)', color: '#ffffff', margin: 0, letterSpacing: '0.04em', lineHeight: 1 }
const panel    = { border: '1px solid #1a1a1a', background: '#0a0a0a', padding: '1.75rem' }
const panelLabel = { ...eyebrow, margin: '0 0 1.5rem' }
const inputStyle = { background: '#000', border: '1px solid #333', color: '#fff', fontFamily: '"Courier New",monospace', fontSize: '0.7rem', letterSpacing: '0.08em', padding: '0.6rem 0.85rem', width: '100%', outline: 'none' }
const td       = { padding: '0.6rem 1rem', whiteSpace: 'nowrap', verticalAlign: 'middle' }
