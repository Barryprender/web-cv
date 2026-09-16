// Applies the stored theme override before the first paint.
//
// Deliberately a blocking classic script in <head>, and deliberately not part
// of the main.js module graph: a module is always deferred, so it runs after
// the document has already painted. A visitor whose OS is light but who chose
// dark therefore saw one frame of the light theme first. With cross-document
// view transitions that frame is what gets captured into the outgoing
// snapshot, so the flash repeated on every navigation instead of only on the
// first load.
//
// It is placed above the stylesheet link for the same reason: a classic script
// that follows a <link rel=stylesheet> waits for that sheet to load before it
// executes, which would hand the flash straight back.
//
// The write side lives in components/cv-theme-toggle.js. The storage key below
// is the one thing the two files share; change it in both or in neither.
(() => {
  try {
    const stored = localStorage.getItem('cv-theme');
    if (stored === 'light' || stored === 'dark') {
      document.documentElement.dataset.theme = stored;
    }
  } catch {
    // Private mode, or storage disabled. Fall through to the OS preference,
    // which the stylesheet handles on its own.
  }
})();
