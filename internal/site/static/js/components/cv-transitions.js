// Gives cross-document view transitions a direction.
//
// @view-transition in the stylesheet already turns navigations into animated
// transitions on its own. This module only decides which way the content
// slides, by comparing the two routes against the order they appear in the nav
// and stamping data-vt on the document root. The stylesheet reads that
// attribute; if this file never runs, the transition falls back to the default
// cross-fade and nothing breaks.
//
// Both events matter: pageswap fires on the outgoing document (styling the
// "old" snapshot) and pagereveal on the incoming one (styling the "new"), so
// the attribute has to be set in each.

const NAV_ORDER = ['/', '/experience', '/projects', '/skills', '/contact'];

function position(pathname) {
  // Trailing slashes and unknown routes both fall through to -1, which reads
  // as "no opinion" below.
  const normalised = pathname.length > 1 ? pathname.replace(/\/$/, '') : pathname;
  return NAV_ORDER.indexOf(normalised);
}

function direction(fromPath, toPath) {
  const from = position(fromPath);
  const to = position(toPath);
  if (from === -1 || to === -1 || from === to) return null;
  return to > from ? 'forward' : 'back';
}

function mark(fromPath, toPath) {
  const dir = direction(fromPath, toPath);
  if (dir) {
    document.documentElement.dataset.vt = dir;
  } else {
    delete document.documentElement.dataset.vt;
  }
}

// Leaving: the destination is the entry being activated.
window.addEventListener('pageswap', (event) => {
  if (!event.viewTransition) return;
  const destination = event.activation?.entry?.url;
  if (destination) mark(location.pathname, new URL(destination).pathname);
});

// Arriving: the origin is the entry navigated away from.
window.addEventListener('pagereveal', (event) => {
  if (!event.viewTransition) return;
  const origin = navigation?.activation?.from?.url;
  if (origin) mark(new URL(origin).pathname, location.pathname);
});
