// Enhances the server-rendered quick-jump panel.
//
// The panel itself is a native popover: the nav button opens it via
// popovertarget, the browser handles the top layer, light dismiss and Escape,
// and every entry inside is an ordinary link. All of that works with this file
// absent. What it adds is the keyboard shortcut, a filter field, and arrow-key
// navigation between results.

const PALETTE_ID = 'command-palette';

function isMac() {
  const platform = navigator.userAgentData?.platform ?? navigator.platform ?? '';
  return /mac|iphone|ipad/i.test(platform);
}

function init() {
  const palette = document.getElementById(PALETTE_ID);
  if (!palette || typeof palette.showPopover !== 'function') return;

  const list = palette.querySelector('.palette-list');
  const empty = palette.querySelector('.palette-empty');
  const items = [...palette.querySelectorAll('.palette-item')];
  const groups = [...palette.querySelectorAll('.palette-group')];
  if (!list || !items.length) return;

  // The shortcut hint is rendered as the Mac glyph; correct it elsewhere.
  if (!isMac()) {
    for (const key of document.querySelectorAll('.palette-key')) {
      key.textContent = 'Ctrl K';
    }
  }

  // The filter field is created here rather than in the template, so it never
  // appears as a dead input when this script has not run.
  const search = document.createElement('input');
  search.type = 'search';
  search.className = 'palette-search';
  search.placeholder = isMac() ? 'Jump to…  ⌘K' : 'Jump to…  Ctrl K';
  search.setAttribute('aria-label', 'Filter destinations');
  search.autocomplete = 'off';
  list.before(search);

  const visible = () => items.filter((item) => !item.hidden);

  function filter(query) {
    const needle = query.trim().toLowerCase();
    for (const item of items) {
      item.hidden = needle !== '' && !item.textContent.toLowerCase().includes(needle);
    }
    // Hide a group label when everything under it is filtered out.
    for (const group of groups) {
      let hasVisible = false;
      for (let el = group.nextElementSibling; el && !el.matches('.palette-group'); el = el.nextElementSibling) {
        if (el.matches('.palette-item') && !el.hidden) {
          hasVisible = true;
          break;
        }
      }
      group.hidden = !hasVisible;
    }
    empty.hidden = visible().length > 0;
  }

  function move(step) {
    const options = visible();
    if (!options.length) return;
    const current = options.indexOf(document.activeElement);
    // From the search field, ArrowDown enters the list at the top.
    const next = current === -1
      ? (step > 0 ? 0 : options.length - 1)
      : (current + step + options.length) % options.length;
    options[next].focus();
  }

  palette.addEventListener('keydown', (event) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      move(1);
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      move(-1);
    } else if (event.key === 'Home' && document.activeElement !== search) {
      event.preventDefault();
      visible()[0]?.focus();
    }
  });

  search.addEventListener('input', () => filter(search.value));

  // Reset and focus each time it opens, so it never reopens mid-filter.
  palette.addEventListener('toggle', (event) => {
    if (event.newState !== 'open') return;
    search.value = '';
    filter('');
    // Focus after the popover has been promoted to the top layer.
    requestAnimationFrame(() => search.focus());
  });

  document.addEventListener('keydown', (event) => {
    const accel = isMac() ? event.metaKey : event.ctrlKey;
    if (!accel || event.key.toLowerCase() !== 'k') return;
    event.preventDefault();
    if (palette.matches(':popover-open')) {
      palette.hidePopover();
    } else {
      palette.showPopover();
    }
  });
}

init();
