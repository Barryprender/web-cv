// <cv-nav> progressively enhances a server-rendered nav bar with a mobile
// menu toggle. It renders nothing itself — the markup and active-link
// state (aria-current="page") are set server-side so the nav works and
// is crawlable with JS disabled.
class CvNav extends HTMLElement {
  connectedCallback() {
    const btn = this.querySelector('.nav-mobile-btn');
    const links = this.querySelector('.nav-links');
    if (!btn || !links) return;
 
    btn.addEventListener('click', () => {
      const open = links.classList.toggle('open');
      btn.setAttribute('aria-expanded', String(open));
    });
  }
}
 
customElements.define('cv-nav', CvNav);