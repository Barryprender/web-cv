// <cv-theme-toggle> flips a light/dark override on <html data-theme="...">.
// Default (no attribute) follows the OS via prefers-color-scheme, styled
// in CSS. The choice is remembered per browser in localStorage; failures
// there (private mode, disabled storage) just fall back to in-memory only.
//
// Reading the stored value back is not done here. This module is deferred, so
// applying it on connect painted a light frame first; ../theme.js does it in
// <head> before the first paint instead. This file only writes.
const STORAGE_KEY = 'cv-theme';
 
function writeStored(value) {
  try {
    if (value) localStorage.setItem(STORAGE_KEY, value);
    else localStorage.removeItem(STORAGE_KEY);
  } catch {
    // ignore — theme choice just won't persist this session
  }
}
 
class CvThemeToggle extends HTMLElement {
  connectedCallback() {
    this.innerHTML = '';
    const btn = document.createElement('button');
    btn.className = 'nav-toggle-btn';
    btn.type = 'button';
    this.append(btn);
    this.button = btn;
    this.render();
 
    btn.addEventListener('click', () => {
      const current = document.documentElement.getAttribute('data-theme');
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      const currentlyDark = current ? current === 'dark' : prefersDark;
      const next = currentlyDark ? 'light' : 'dark';
      document.documentElement.setAttribute('data-theme', next);
      writeStored(next);
      this.render();
    });
  }
 
  render() {
    const current = document.documentElement.getAttribute('data-theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const isDark = current ? current === 'dark' : prefersDark;
    const next = isDark ? 'light' : 'dark';
    this.button.textContent = next;
    // The accessible name has to contain the visible label, or a speech-input
    // user saying "click dark" cannot activate it (WCAG 2.5.3, Label in Name).
    // A fixed "Toggle color theme" label did not contain it.
    this.button.setAttribute('aria-label', `Switch to ${next} theme`);
  }
}
 
customElements.define('cv-theme-toggle', CvThemeToggle);