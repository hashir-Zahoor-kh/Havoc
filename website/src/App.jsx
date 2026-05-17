import Navbar from './components/Navbar'
import Hero from './components/Hero'
import Architecture from './components/Architecture'
import Demo from './components/Demo'
import Safety from './components/Safety'
import Capabilities from './components/Capabilities'

const STUB = {
  minHeight: '100vh',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  fontFamily: '"Courier New", Courier, monospace',
  color: '#333',
  fontSize: '0.75rem',
  letterSpacing: '0.1em',
  textTransform: 'uppercase',
  borderBottom: '1px solid #0f0f0f',
}

export default function App() {
  return (
    <div style={{ background: '#000000', color: '#ffffff', minHeight: '100vh' }}>
      <Navbar />
      <div style={{ paddingTop: '56px' }}>
        <Hero />
        <Architecture />
        <Demo />
        <Safety />
        <Capabilities />
        <section id="about" style={STUB}>About — coming in Stage 8</section>
      </div>
    </div>
  )
}
