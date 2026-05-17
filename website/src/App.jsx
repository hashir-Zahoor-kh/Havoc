import Navbar from './components/Navbar'
import Hero from './components/Hero'
import Architecture from './components/Architecture'
import Demo from './components/Demo'
import Safety from './components/Safety'
import Capabilities from './components/Capabilities'
import About from './components/About'


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
        <About />
      </div>
    </div>
  )
}
