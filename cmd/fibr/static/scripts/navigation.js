/**
 * Async loading of a script
 * @param {String} src         URL of script
 * @param {String} integrity   Integrity of script
 * @param {String} crossorigin Crossorigin of script
 * @return Promise when script is either loaded or on error
 */
function resolveScript(src, integrity, crossorigin) {
  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.type = "text/javascript";
    script.src = src;
    script.async = true;
    script.onload = resolve.bind(null, true);
    script.onerror = reject.bind(null, true);

    if (integrity) {
      script.integrity = integrity;
      script.crossOrigin = crossorigin;
    }

    document.querySelector("head").appendChild(script);
  });
}

/**
 * Handle Previous/next.
 */
window.onkeyup = (e) => {
  switch (e.key) {
    case "ArrowLeft":
      goToPrevious();
      break;

    case "ArrowRight":
      goToNext();
      break;

    case "Escape":
      goBack(e);
      break;
  }
};

function goBack(e) {
  const previousHash = document.location.hash;

  if (previousHash) {
    if (typeof abort === "function" && typeof aborter !== "undefined") {
      abort(e);
      return;
    }

    document.location.hash = "";

    if (/success$/gim.test(previousHash)) {
      window.location.reload(true);
    }

    return;
  }

  if (typeof parentPage === "undefined") {
    return;
  }

  window.location.href = parentPage;
}

/**
 * Go to the previous item.
 */
function goToPrevious() {
  if (typeof previousFile === "undefined") {
    return;
  }

  window.location.href = previousFile;
}

/**
 * Go to the next item.
 */
function goToNext() {
  if (typeof nextFile === "undefined") {
    return;
  }

  window.location.href = nextFile;
}

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
