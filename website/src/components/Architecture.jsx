import { useState, useRef, useLayoutEffect, useCallback, useEffect } from 'react'
import { motion } from 'framer-motion'
import { components, techStack } from '../data/mockData'

const DIAGRAM_NODES = [
  { id: 'cli',            gridCol: 1, gridRow: 1 },
  { id: 'control',        gridCol: 2, gridRow: 1 },
  { id: 'kafka-cmd',      gridCol: 3, gridRow: 1 },
  { id: 'agents',         gridCol: 4, gridRow: 1 },
  { id: 'postgres-redis', gridCol: 2, gridRow: 2 },
  { id: 'kafka-res',      gridCol: 4, gridRow: 2 },
  { id: 'recorder',       gridCol: 4, gridRow: 3 },
  { id: 'storage',        gridCol: 4, gridRow: 4 },
]

const EDGES = [
  { from: 'cli',         to: 'control' },
  { from: 'control',     to: 'kafka-cmd' },
  { from: 'kafka-cmd',   to: 'agents' },
  { from: 'control',     to: 'postgres-redis' },
  { from: 'agents',      to: 'kafka-res' },
  { from: 'kafka-res',   to: 'recorder' },
  { from: 'recorder',    to: 'storage' },
]

const compMap = Object.fromEntries(components.map(c => [c.id, c]))

const MOBILE_ORDER = ['cli', 'control', 'kafka-cmd', 'agents', 'kafka-res', 'recorder', 'storage', 'postgres-redis']

export default function Architecture() {
  const [selected, setSelected] = useState(null)
  const [arrows, setArrows] = useState([])
  const [svgSize, setSvgSize] = useState({ w: 0, h: 0 })
  const [isMobile, setIsMobile] = useState(false)
  const nodeRefs = useRef({})
  const containerRef = useRef(null)

  useEffect(() => {
    const check = () => setIsMobile(window.innerWidth < 900)
    check()
    window.addEventListener('resize', check)
    return () => window.removeEventListener('resize', check)
  }, [])

  const computeArrows = useCallback(() => {
    const container = containerRef.current
    if (!container) return
    const cRect = container.getBoundingClientRect()
    setSvgSize({ w: cRect.width, h: cRect.height })

    const newArrows = EDGES.map(({ from, to }, idx) => {
      const fromEl = nodeRefs.current[from]
      const toEl = nodeRefs.current[to]
      if (!fromEl || !toEl) return null

      const fRect = fromEl.getBoundingClientRect()
      const tRect = toEl.getBoundingClientRect()

      const fCx = fRect.left - cRect.left + fRect.width / 2
      const fCy = fRect.top - cRect.top + fRect.height / 2
      const tCx = tRect.left - cRect.left + tRect.width / 2
      const tCy = tRect.top - cRect.top + tRect.height / 2

      let x1, y1, x2, y2
      if (Math.abs(tCx - fCx) > Math.abs(tCy - fCy)) {
        x1 = tCx > fCx ? fRect.right - cRect.left : fRect.left - cRect.left
        y1 = fCy
        x2 = tCx > fCx ? tRect.left - cRect.left : tRect.right - cRect.left
        y2 = tCy
      } else {
        x1 = fCx
        y1 = tCy > fCy ? fRect.bottom - cRect.top : fRect.top - cRect.top
        x2 = tCx
        y2 = tCy > fCy ? tRect.top - cRect.top : tRect.bottom - cRect.top
      }

      const len = Math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2)
      return { from, to, x1, y1, x2, y2, len, delay: idx * 0.18 }
    }).filter(Boolean)

    setArrows(newArrows)
  }, [])

  useLayoutEffect(() => {
    if (isMobile) return
    computeArrows()
    const ro = new ResizeObserver(computeArrows)
    if (containerRef.current) ro.observe(containerRef.current)
    return () => ro.disconnect()
  }, [computeArrows, isMobile])

  const selectedComp = selected ? compMap[selected] : null

  return (
    <section id="architecture" style={{ background: '#000000', padding: '7rem 2rem', borderBottom: '1px solid #0f0f0f' }}>
      <div style={{ maxWidth: '1200px', margin: '0 auto' }}>

        {/* Section header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.3, ease: 'easeOut' }}
          style={{ marginBottom: '4rem' }}
        >
          <p style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.7rem', letterSpacing: '0.18em', color: '#caff00', textTransform: 'uppercase', margin: '0 0 0.6rem' }}>
            // HOW IT WORKS
          </p>
          <h2 style={{ fontFamily: '"Bebas Neue",cursive', fontSize: 'clamp(3rem,8vw,5.5rem)', color: '#ffffff', margin: 0, letterSpacing: '0.04em', lineHeight: 1 }}>
            ARCHITECTURE
          </h2>
        </motion.div>

        {/* Diagram */}
        {!isMobile ? (
          <div ref={containerRef} style={{ position: 'relative', marginBottom: '2.5rem' }}>
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(4, 1fr)',
              gridTemplateRows: 'repeat(4, auto)',
              gap: '3rem 1.5rem',
            }}>
              {DIAGRAM_NODES.map(({ id, gridCol, gridRow }) => {
                const comp = compMap[id]
                if (!comp) return null
                return (
                  <div
                    key={id}
                    ref={el => { nodeRefs.current[id] = el }}
                    style={{ gridColumn: gridCol, gridRow: gridRow }}
                  >
                    <NodeCard
                      comp={comp}
                      isSelected={selected === id}
                      onClick={() => setSelected(selected === id ? null : id)}
                    />
                  </div>
                )
              })}
            </div>

            {/* SVG arrow overlay */}
            <svg
              style={{ position: 'absolute', top: 0, left: 0, pointerEvents: 'none', overflow: 'visible' }}
              width={svgSize.w}
              height={svgSize.h}
            >
              <defs>
                <marker id="ah" markerWidth="7" markerHeight="7" refX="5" refY="3.5" orient="auto">
                  <polygon points="0,0 7,3.5 0,7" fill="#2a2a2a" />
                </marker>
              </defs>
              {arrows.map(({ from, to, x1, y1, x2, y2, len, delay }) => (
                <g key={`${from}-${to}`}>
                  <line x1={x1} y1={y1} x2={x2} y2={y2} stroke="#1a1a1a" strokeWidth="1" markerEnd="url(#ah)" />
                  <line
                    x1={x1} y1={y1} x2={x2} y2={y2}
                    stroke="#caff00"
                    strokeWidth="1.5"
                    strokeDasharray="5 23"
                    style={{
                      animation: `flow-along 1.6s linear infinite`,
                      animationDelay: `${delay}s`,
                    }}
                  />
                </g>
              ))}
            </svg>
          </div>
        ) : (
          <MobileFlow onSelect={id => setSelected(selected === id ? null : id)} selected={selected} />
        )}

        {/* Description panel */}
        {selectedComp && (
          <motion.div
            key={selectedComp.id}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
            style={{
              border: '1px solid #222',
              background: '#0a0a0a',
              padding: '1.5rem 1.75rem',
              marginBottom: '3rem',
              display: 'flex',
              gap: '2rem',
              alignItems: 'flex-start',
            }}
          >
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '0.8rem', flexWrap: 'wrap' }}>
                <span style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.78rem', letterSpacing: '0.12em', color: '#ffffff', textTransform: 'uppercase' }}>
                  {selectedComp.label.replace(/\n/g, ' ')}
                </span>
                <span style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.6rem', letterSpacing: '0.1em', color: '#caff00', border: '1px solid #333', padding: '0.15rem 0.5rem', textTransform: 'uppercase' }}>
                  {selectedComp.tag}
                </span>
              </div>
              <p style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.76rem', lineHeight: 1.75, color: '#666666', margin: 0 }}>
                {selectedComp.description}
              </p>
            </div>
            <button
              onClick={() => setSelected(null)}
              style={{ background: 'none', border: 'none', color: '#444', cursor: 'pointer', fontFamily: '"Courier New",Courier,monospace', fontSize: '0.65rem', letterSpacing: '0.1em', flexShrink: 0, padding: 0 }}
              onMouseEnter={e => { e.currentTarget.style.color = '#fff' }}
              onMouseLeave={e => { e.currentTarget.style.color = '#444' }}
            >
              [CLOSE]
            </button>
          </motion.div>
        )}

        {/* Tech stack pills */}
        <TechStackPills />
      </div>
    </section>
  )
}

function NodeCard({ comp, isSelected, onClick }) {
  const [hovered, setHovered] = useState(false)
  const active = isSelected || hovered

  return (
    <button
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: '100%',
        background: '#0f0f0f',
        border: `1px solid ${active ? '#caff00' : '#222'}`,
        padding: '1rem 1rem 0.85rem',
        cursor: 'pointer',
        textAlign: 'left',
        position: 'relative',
        transition: 'border-color 150ms ease-out',
        display: 'block',
      }}
    >
      {/* Category tag */}
      <span style={{
        position: 'absolute',
        top: '0.55rem',
        right: '0.55rem',
        fontFamily: '"Courier New",Courier,monospace',
        fontSize: '0.55rem',
        letterSpacing: '0.1em',
        textTransform: 'uppercase',
        color: '#444',
      }}>
        {comp.tag}
      </span>

      {/* Component name */}
      <span style={{
        display: 'block',
        fontFamily: '"Courier New",Courier,monospace',
        fontSize: '0.72rem',
        letterSpacing: '0.1em',
        textTransform: 'uppercase',
        color: active ? '#caff00' : '#ffffff',
        lineHeight: 1.4,
        paddingRight: '2.5rem',
        transition: 'color 150ms ease-out',
        whiteSpace: 'pre-line',
      }}>
        {comp.label}
      </span>

      {/* Click hint */}
      <span style={{
        display: 'block',
        fontFamily: '"Courier New",Courier,monospace',
        fontSize: '0.55rem',
        letterSpacing: '0.08em',
        color: '#333',
        marginTop: '0.4rem',
        transition: 'color 150ms ease-out',
      }}>
        {isSelected ? '▲ HIDE' : '▼ DETAILS'}
      </span>
    </button>
  )
}

function MobileFlow({ onSelect, selected }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '0', marginBottom: '2.5rem' }}>
      {MOBILE_ORDER.map((id, i) => {
        const comp = compMap[id]
        if (!comp) return null
        const isLast = i === MOBILE_ORDER.length - 1
        return (
          <div key={id}>
            <button
              onClick={() => onSelect(id)}
              style={{
                width: '100%',
                background: '#0f0f0f',
                border: `1px solid ${selected === id ? '#caff00' : '#222'}`,
                padding: '0.9rem 1rem',
                cursor: 'pointer',
                textAlign: 'left',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <span style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.7rem', letterSpacing: '0.1em', color: selected === id ? '#caff00' : '#fff', textTransform: 'uppercase' }}>
                {comp.label.replace(/\n/g, ' ')}
              </span>
              <span style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.55rem', color: '#555', letterSpacing: '0.1em' }}>
                {comp.tag}
              </span>
            </button>
            {!isLast && (
              <div style={{ display: 'flex', justifyContent: 'center', padding: '0.3rem 0', color: '#2a2a2a', fontFamily: 'monospace', fontSize: '0.8rem' }}>↓</div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function TechStackPills() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 16 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.3, ease: 'easeOut', delay: 0.1 }}
    >
      <p style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.65rem', letterSpacing: '0.15em', color: '#444', textTransform: 'uppercase', margin: '0 0 1.5rem' }}>
        // TECH STACK
      </p>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
        {Object.entries(techStack).map(([category, items]) => (
          <div key={category} style={{ display: 'flex', alignItems: 'baseline', gap: '1rem', flexWrap: 'wrap' }}>
            <span style={{ fontFamily: '"Courier New",Courier,monospace', fontSize: '0.6rem', letterSpacing: '0.14em', color: '#444', textTransform: 'uppercase', minWidth: '120px', flexShrink: 0 }}>
              {category}
            </span>
            <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
              {items.map(item => (
                <span
                  key={item}
                  style={{
                    fontFamily: '"Courier New",Courier,monospace',
                    fontSize: '0.65rem',
                    letterSpacing: '0.08em',
                    color: '#888',
                    border: '1px solid #222',
                    padding: '0.25rem 0.65rem',
                    textTransform: 'uppercase',
                  }}
                >
                  {item}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </motion.div>
  )
}
