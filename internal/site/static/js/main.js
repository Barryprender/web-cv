// Registers every custom element. Loaded once, site-wide, as a module —
// no bundler, no external dependencies.
import './components/cv-nav.js';
import './components/cv-theme-toggle.js';
import './components/cv-timeline.js';
import './components/cv-contact-form.js';
import './components/cv-filter.js';
import './components/cv-palette.js';
import './components/cv-transitions.js';

// Developer console greeting. Recruiters do not open DevTools; the engineers
// who review this site do, and this is the one place to talk to them directly.
// Styles are literal hex rather than the stylesheet's oklch tokens: the console
// is outside the document, so var(--accent) resolves to nothing there.
{
  const es = document.documentElement.lang.startsWith('es');
  const at = es ? '/es' : '';
  const accent = 'color:#1f9d63;font-weight:600;font-size:12px;font-family:monospace';
  const dim = 'color:#7c8796;font-size:12px;font-family:monospace';

  console.log(
    '%c⬡ Barry Prendergast',
    'color:#1f9d63;font-size:18px;font-weight:800;letter-spacing:-0.02em;font-family:monospace',
  );
  console.log(
    `%c${es ? 'Ingeniero Full-Stack Senior' : 'Senior Full-Stack Engineer'}\n` +
      'Go standard library · native Web Components · zero dependencies',
    'color:#475569;font-size:12px;line-height:1.8;font-family:monospace',
  );
  console.log(
    `%c→ ${at}/experience%c   ${es ? 'Quince años, filtrable' : 'Fifteen years, filterable'}\n` +
      `%c→ ${at}/contact%c   ${es ? 'Escríbeme' : 'Say hello'}\n` +
      '%c→ github.com/Barryprender%c   ' +
      `${es ? 'El código de este sitio' : "This site's source"}`,
    accent,
    dim,
    accent,
    dim,
    accent,
    dim,
  );
}
