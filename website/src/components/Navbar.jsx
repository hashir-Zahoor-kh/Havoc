const NAV_LINKS = [
  { label: 'OVERVIEW', href: '#overview' },
  { label: 'ARCHITECTURE', href: '#architecture' },
  { label: 'DEMO', href: '#demo' },
  { label: 'SAFETY', href: '#safety' },
  { label: 'ABOUT', href: '#about' },
]

export default function Navbar() {
  const handleNav = (e, href) => {
    e.preventDefault()
    const el = document.querySelector(href)
    if (el) el.scrollIntoView({ behavior: 'smooth' })
  }

  return (
    <header style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      zIndex: 100,
      background: '#000000',
      borderBottom: '1px solid #1a1a1a',
    }}>
      <nav style={{
        maxWidth: '1400px',
        margin: '0 auto',
        padding: '0 2rem',
        height: '56px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: '2rem',
      }}>
        {/* Wordmark */}
        <a
          href="#"
          onClick={e => { e.preventDefault(); window.scrollTo({ top: 0, behavior: 'smooth' }) }}
          style={{
            fontFamily: '"Bebas Neue", cursive',
            fontSize: '1.85rem',
            color: '#ffffff',
            textDecoration: 'none',
            letterSpacing: '0.06em',
            lineHeight: 1,
            flexShrink: 0,
          }}
        >
          HAVOC
        </a>

        {/* Center nav links */}
        <ul style={{
          display: 'flex',
          alignItems: 'center',
          gap: '2rem',
          listStyle: 'none',
          margin: 0,
          padding: 0,
          flex: 1,
          justifyContent: 'center',
        }}>
          {NAV_LINKS.map(({ label, href }) => (
            <li key={label}>
              <NavLink label={label} href={href} onClick={handleNav} />
            </li>
          ))}
        </ul>

        {/* Right side */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '1.5rem', flexShrink: 0 }}>
          {/* Status indicator */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <span
              className="pulse-dot"
              style={{
                display: 'inline-block',
                width: '7px',
                height: '7px',
                background: '#caff00',
                borderRadius: '50%',
                flexShrink: 0,
              }}
            />
            <span style={{
              fontFamily: '"Courier New", Courier, monospace',
              fontSize: '0.65rem',
              letterSpacing: '0.12em',
              color: '#666666',
              textTransform: 'uppercase',
              whiteSpace: 'nowrap',
            }}>
              PLATFORM HEALTHY
            </span>
          </div>

          {/* GitHub button */}
          <GithubButton />
        </div>
      </nav>

      {/* Mobile nav — horizontal scroll */}
      <MobileNav links={NAV_LINKS} onNav={handleNav} />
    </header>
  )
}

function NavLink({ label, href, onClick }) {
  return (
    <a
      href={href}
      onClick={e => onClick(e, href)}
      style={{
        fontFamily: '"Courier New", Courier, monospace',
        fontSize: '0.7rem',
        letterSpacing: '0.13em',
        textTransform: 'uppercase',
        color: '#888888',
        textDecoration: 'none',
        transition: 'color 150ms ease-out',
        whiteSpace: 'nowrap',
      }}
      onMouseEnter={e => { e.currentTarget.style.color = '#ffffff' }}
      onMouseLeave={e => { e.currentTarget.style.color = '#888888' }}
    >
      {label}
    </a>
  )
}

function GithubButton() {
  return (
    <a
      href="https://github.com/hashirzahoor"
      target="_blank"
      rel="noopener noreferrer"
      style={{
        fontFamily: '"Courier New", Courier, monospace',
        fontSize: '0.65rem',
        letterSpacing: '0.13em',
        textTransform: 'uppercase',
        color: '#ffffff',
        textDecoration: 'none',
        border: '1px solid #333333',
        padding: '0.45rem 0.85rem',
        transition: 'color 150ms ease-out, border-color 150ms ease-out',
        whiteSpace: 'nowrap',
        display: 'inline-block',
      }}
      onMouseEnter={e => {
        e.currentTarget.style.color = '#caff00'
        e.currentTarget.style.borderColor = '#caff00'
      }}
      onMouseLeave={e => {
        e.currentTarget.style.color = '#ffffff'
        e.currentTarget.style.borderColor = '#333333'
      }}
    >
      VIEW ON GITHUB
    </a>
  )
}

function MobileNav({ links, onNav }) {
  return (
    <div style={{
      display: 'none',
      overflowX: 'auto',
      borderTop: '1px solid #0f0f0f',
      padding: '0.6rem 1.5rem',
      gap: '1.5rem',
      scrollbarWidth: 'none',
    }}
    className="mobile-nav"
    >
      {links.map(({ label, href }) => (
        <a
          key={label}
          href={href}
          onClick={e => onNav(e, href)}
          style={{
            fontFamily: '"Courier New", Courier, monospace',
            fontSize: '0.65rem',
            letterSpacing: '0.13em',
            textTransform: 'uppercase',
            color: '#888',
            textDecoration: 'none',
            whiteSpace: 'nowrap',
          }}
        >
          {label}
        </a>
      ))}
    </div>
  )
}
