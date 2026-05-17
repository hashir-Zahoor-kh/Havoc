import Navbar from './components/Navbar'

const SECTION_STYLE = {
  minHeight: '100vh',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  fontFamily: '"Courier New", Courier, monospace',
  color: '#444',
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
        <section id="overview" style={SECTION_STYLE}>Overview — coming in Stage 3</section>
        <section id="architecture" style={SECTION_STYLE}>Architecture — coming in Stage 4</section>
        <section id="demo" style={SECTION_STYLE}>Demo — coming in Stage 5</section>
        <section id="safety" style={SECTION_STYLE}>Safety — coming in Stage 6</section>
        <section id="about" style={SECTION_STYLE}>About — coming in Stage 8</section>
      </div>
    </div>
  )
}
