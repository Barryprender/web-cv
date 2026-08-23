// <cv-timeline> wraps server-rendered .timeline-item entries (kept in the
// light DOM so the experience text stays crawlable) and adds two behaviors:
// scroll-reveal via IntersectionObserver, and click-to-expand bullets.
//
// The reveal animation is strictly additive: the entries are visible in CSS
// by default, and this element opts into the hidden-then-fade-in state by
// setting data-reveal on itself. If this module never loads, or the browser
// has no IntersectionObserver, or the visitor asked for reduced motion, the
// attribute is never set and the content simply stays visible.
class CvTimeline extends HTMLElement {
  connectedCallback() {
    const items = Array.from(this.querySelectorAll('.timeline-item'));
    if (!items.length) return;

    this.#enableReveal(items);
    this.#enableToggles();
  }

  #enableReveal(items) {
    if (!('IntersectionObserver' in window)) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

    this.dataset.reveal = '';

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            entry.target.classList.add('in-view');
            observer.unobserve(entry.target);
          }
        }
      },
      { threshold: 0.15 }
    );
    items.forEach((item) => observer.observe(item));
  }

  #enableToggles() {
    this.addEventListener('click', (event) => {
      const toggle = event.target.closest('.timeline-toggle');
      if (!toggle) return;
      const item = toggle.closest('.timeline-item');
      const expanded = item.classList.toggle('expanded');
      // Rewrite only the label span, so the visually-hidden company name the
      // template puts alongside it stays part of the accessible name. Falls
      // back to the button itself if that span is ever removed.
      const label = toggle.querySelector('.timeline-toggle-label') ?? toggle;
      label.textContent = expanded ? '- collapse' : '+ details';
      toggle.setAttribute('aria-expanded', String(expanded));
    });
  }
}

customElements.define('cv-timeline', CvTimeline);
