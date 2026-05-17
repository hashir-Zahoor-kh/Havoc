import { useEffect, useRef, useCallback } from 'react'

const COLS = 14
const ROWS = 8
const NODE_R = 2.2

export default function NodeGrid() {
  const canvasRef = useRef(null)
  const animRef = useRef(null)
  const nodesRef = useRef([])
  const chaosNodeRef = useRef(null)
  const chaosPhaseRef = useRef('idle') // idle | dying | dead | recovering
  const chaosTimerRef = useRef(0)

  const buildNodes = useCallback((w, h) => {
    const nodes = []
    const padX = w * 0.06
    const padY = h * 0.12
    const spacingX = (w - padX * 2) / (COLS - 1)
    const spacingY = (h - padY * 2) / (ROWS - 1)
    for (let r = 0; r < ROWS; r++) {
      for (let c = 0; c < COLS; c++) {
        nodes.push({
          x: padX + c * spacingX,
          y: padY + r * spacingY,
          opacity: 0.12 + Math.random() * 0.08,
          index: r * COLS + c,
        })
      }
    }
    return nodes
  }, [])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')

    const resize = () => {
      canvas.width = canvas.offsetWidth
      canvas.height = canvas.offsetHeight
      nodesRef.current = buildNodes(canvas.width, canvas.height)
    }
    resize()
    const ro = new ResizeObserver(resize)
    ro.observe(canvas)

    let lastTime = 0
    const CHAOS_INTERVAL = 4200

    const draw = (time) => {
      animRef.current = requestAnimationFrame(draw)
      const delta = time - lastTime
      lastTime = time

      const { width: w, height: h } = canvas
      ctx.clearRect(0, 0, w, h)

      const nodes = nodesRef.current
      if (!nodes.length) return

      // advance chaos state machine
      chaosTimerRef.current += delta
      if (chaosPhaseRef.current === 'idle' && chaosTimerRef.current > CHAOS_INTERVAL) {
        chaosTimerRef.current = 0
        chaosPhaseRef.current = 'dying'
        // pick a node that isn't on the very edge
        const inner = nodes.filter(n => {
          const col = n.index % COLS
          const row = Math.floor(n.index / COLS)
          return col > 1 && col < COLS - 2 && row > 0 && row < ROWS - 1
        })
        chaosNodeRef.current = inner[Math.floor(Math.random() * inner.length)]
      } else if (chaosPhaseRef.current === 'dying' && chaosTimerRef.current > 600) {
        chaosTimerRef.current = 0
        chaosPhaseRef.current = 'dead'
      } else if (chaosPhaseRef.current === 'dead' && chaosTimerRef.current > 1200) {
        chaosTimerRef.current = 0
        chaosPhaseRef.current = 'recovering'
      } else if (chaosPhaseRef.current === 'recovering' && chaosTimerRef.current > 700) {
        chaosTimerRef.current = 0
        chaosPhaseRef.current = 'idle'
        chaosNodeRef.current = null
      }

      const chaosIdx = chaosNodeRef.current?.index ?? -1

      // draw edges
      for (let i = 0; i < nodes.length; i++) {
        const a = nodes[i]
        const col = i % COLS
        const row = Math.floor(i / COLS)
        const neighbors = []
        if (col < COLS - 1) neighbors.push(nodes[i + 1])
        if (row < ROWS - 1) neighbors.push(nodes[i + COLS])

        for (const b of neighbors) {
          const isChaosEdge = a.index === chaosIdx || b.index === chaosIdx
          ctx.beginPath()
          ctx.moveTo(a.x, a.y)
          ctx.lineTo(b.x, b.y)

          if (isChaosEdge) {
            const phase = chaosPhaseRef.current
            if (phase === 'dying') {
              ctx.strokeStyle = 'rgba(255,51,51,0.18)'
            } else if (phase === 'dead') {
              ctx.strokeStyle = 'rgba(255,51,51,0.06)'
            } else if (phase === 'recovering') {
              ctx.strokeStyle = 'rgba(202,255,0,0.12)'
            } else {
              ctx.strokeStyle = 'rgba(255,255,255,0.06)'
            }
          } else {
            ctx.strokeStyle = 'rgba(255,255,255,0.06)'
          }
          ctx.lineWidth = 0.5
          ctx.stroke()
        }
      }

      // draw nodes
      for (const node of nodes) {
        const isChaos = node.index === chaosIdx
        ctx.beginPath()
        ctx.arc(node.x, node.y, NODE_R, 0, Math.PI * 2)

        if (isChaos) {
          const phase = chaosPhaseRef.current
          const t = chaosTimerRef.current
          if (phase === 'dying') {
            const pulse = 0.5 + 0.5 * Math.sin(t * 0.02)
            ctx.fillStyle = `rgba(255,51,51,${0.6 + pulse * 0.4})`
          } else if (phase === 'dead') {
            ctx.fillStyle = 'rgba(255,51,51,0.15)'
          } else if (phase === 'recovering') {
            const pulse = 0.5 + 0.5 * Math.sin(t * 0.015)
            ctx.fillStyle = `rgba(202,255,0,${0.3 + pulse * 0.3})`
          } else {
            ctx.fillStyle = `rgba(255,255,255,${node.opacity})`
          }
        } else {
          ctx.fillStyle = `rgba(255,255,255,${node.opacity})`
        }
        ctx.fill()
      }
    }

    animRef.current = requestAnimationFrame(draw)
    return () => {
      cancelAnimationFrame(animRef.current)
      ro.disconnect()
    }
  }, [buildNodes])

  return (
    <canvas
      ref={canvasRef}
      style={{
        position: 'absolute',
        inset: 0,
        width: '100%',
        height: '100%',
        opacity: 0.9,
        pointerEvents: 'none',
      }}
    />
  )
}
