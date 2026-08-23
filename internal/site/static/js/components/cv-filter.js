// <cv-filter> turns the stack tags inside it into a cross-filter for the whole
// list: pick "Go" and every entry that does not use Go recedes.
//
// The server renders the tags as inert <span> labels, which is the correct
// no-JS result: they read as metadata and nothing suggests they are clickable.
// This element upgrades them to real <button>s only once it has run, so the
// affordance never appears without the behaviour behind it.
//
// Filtering is presentational, so entries are dimmed rather than removed: the
// full history stays on the page and in the accessibility tree, and the
// filtered-out roles are still readable. Only the visual weight changes.

const MUTED = 'is-muted';

class CvFilter extends HTMLElement {
  #active = null;
  #buttons = [];
  #items = [];
  #status = null;

  connectedCallback() {
    const selector = this.dataset.item;
    if (!selector) return;

    this.#items = [...this.querySelectorAll(selector)];
    const tags = [...this.querySelectorAll('.tag[data-stack]')];
    if (!this.#items.length || !tags.length) return;

    this.#buttons = tags.map((tag) => this.#upgrade(tag));
    this.#status = this.#addStatus();
    this.dataset.ready = '';

    this.addEventListener('click', (event) => {
      const button = event.target.closest('.tag-filter');
      if (!button || !this.contains(button)) return;
      this.#apply(button.dataset.stack === this.#active ? null : button.dataset.stack);
    });
  }

  // Swap the span for a button, carrying the label and stack value across.
  #upgrade(tag) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `${tag.className} tag-filter`;
    button.dataset.stack = tag.dataset.stack;
    button.setAttribute('aria-pressed', 'false');
    button.textContent = tag.textContent;
    tag.replaceWith(button);
    return button;
  }

  #addStatus() {
    const status = document.createElement('p');
    status.className = 'filter-status';
    status.setAttribute('role', 'status');
    status.textContent = this.#hint();
    this.prepend(status);
    return status;
  }

  #hint() {
    return `Pick a tag to filter by stack. All ${this.#items.length} ${this.#noun(this.#items.length)} shown.`;
  }

  #noun(n) {
    const noun = this.dataset.noun ?? 'entry';
    return n === 1 ? noun : `${noun}s`;
  }

  #apply(stack) {
    // View Transitions make the reflow read as one movement instead of a jump.
    // Without support, or under reduced motion, the change is simply instant.
    const update = () => this.#render(stack);
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (document.startViewTransition && !reduced) {
      document.startViewTransition(update);
    } else {
      update();
    }
  }

  #render(stack) {
    this.#active = stack;

    for (const button of this.#buttons) {
      button.setAttribute('aria-pressed', String(button.dataset.stack === stack));
    }

    let shown = 0;
    for (const item of this.#items) {
      const matches = stack === null || !!item.querySelector(`.tag-filter[data-stack="${CSS.escape(stack)}"]`);
      item.classList.toggle(MUTED, !matches);
      if (matches) shown += 1;
    }

    this.toggleAttribute('data-filtered', stack !== null);
    this.#status.textContent = stack === null
      ? this.#hint()
      : `${shown} of ${this.#items.length} ${this.#noun(this.#items.length)} use ${stack}. Pick the tag again to clear.`;
  }
}

customElements.define('cv-filter', CvFilter);
