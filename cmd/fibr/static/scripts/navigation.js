/**
 * Async loading of a script
 * @param {String} src         URL of script
 * @param {String} integrity   Integrity of script
 * @param {String} crossorigin Crossorigin of script
 * @return Promise when script is either loaded or on error
 */
function resolveScript(src, integrity, crossorigin) {
  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.type = 'text/javascript';
    script.src = src;
    script.async = true;
    script.onload = resolve.bind(null, true);
    script.onerror = reject.bind(null, true);

    if (integrity) {
      script.integrity = integrity;
      script.crossOrigin = crossorigin;
    }

    document.querySelector('head').appendChild(script);
  });
}

/**
 * Handle Previous/next.
 */
window.onkeyup = (e) => {
  switch (e.key) {
    case 'ArrowLeft':
      goToPrevious();
      break;

    case 'ArrowRight':
      goToNext();
      break;

    case 'Escape':
      if (typeof abort === 'function') {
        abort(e);
      } else {
        goBack();
      }
      break;
  }
};

/**
 * Remove all child and append given one.
 * @param {Element} element Element to clear
 * @param {Element} newContent Element to put in place
 */
function replaceContent(element, newContent) {
  while (element.firstChild) {
    element.removeChild(element.firstChild);
  }

  if (newContent) {
    element.appendChild(newContent);
  }
}
