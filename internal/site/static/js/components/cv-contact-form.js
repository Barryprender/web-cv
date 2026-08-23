// <cv-contact-form> wraps a plain <form method="post" action="/contact">
// and submits it via fetch when JS is available, showing inline status
// instead of a full page reload. With JS disabled the form still works
// exactly as a normal POST against the same endpoint, and the server
// renders the result into the same .form-status element on the redirect.
class CvContactForm extends HTMLElement {
  connectedCallback() {
    const form = this.querySelector('form');
    if (!form) return;

    // The server already rendered this element (carrying any no-JS result).
    // Reuse it rather than appending a second status line; only create one
    // if the markup ever drifts.
    let status = form.querySelector('.form-status');
    if (!status) {
      status = document.createElement('p');
      status.className = 'form-status';
      status.setAttribute('role', 'status');
      form.append(status);
    }

    const submitBtn = form.querySelector('.submit-btn');

    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      status.textContent = 'Sending…';
      status.className = 'form-status';
      if (submitBtn) submitBtn.disabled = true;

      try {
        const res = await fetch(form.action, {
          method: 'POST',
          // URLSearchParams sends application/x-www-form-urlencoded, matching
          // what the plain no-JS form POST sends. FormData would send
          // multipart/form-data, which for a text-only form buys nothing and
          // pushes the server onto its 32MB multipart buffering path.
          body: new URLSearchParams(new FormData(form)),
          headers: { Accept: 'application/json' },
        });
        if (!res.ok) {
          // The server explains refusals precisely: which field is missing,
          // how long to wait, how big is too big. Show that rather than one
          // generic failure line that guesses wrong.
          const detail = await res.json().catch(() => null);
          throw new Error(detail?.error ?? '');
        }
        status.textContent = 'Message sent. I will reply soon.';
        status.className = 'form-status ok';
        form.reset();
      } catch (error) {
        status.textContent = error.message
          || 'Could not reach the server. Email me directly instead.';
        status.className = 'form-status err';
      } finally {
        if (submitBtn) submitBtn.disabled = false;
      }
    });
  }
}

customElements.define('cv-contact-form', CvContactForm);
