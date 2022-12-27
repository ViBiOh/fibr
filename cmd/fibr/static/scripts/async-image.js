// from https://developers.google.com/speed/webp/faq#how_can_i_detect_browser_support_for_webp
function isWebPCompatible() {
  const animatedImage =
    'UklGRlIAAABXRUJQVlA4WAoAAAASAAAAAAAAAAAAQU5JTQYAAAD/////AABBTk1GJgAAAAAAAAAAAAAAAAAAAGQAAABWUDhMDQAAAC8AAAAQBxAREYiI/gcA';

  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => {
      if (image.width > 0 && image.height > 0) {
        resolve();
      } else {
        reject();
      }
    };

    image.onerror = reject.bind(null, true);
    image.src = `data:image/webp;base64,${animatedImage}`;
  });
}

function appendChunk(source, chunk) {
  const output = new Uint8Array(source.length + chunk.length);

  output.set(source, 0);
  output.set(chunk, source.length);

  return output;
}

function findIndexEscapeSequence(escapeSequence, content) {
  let escapePosition = 0;

  for (let i = 0; i < content.length; i++) {
    if (content[i] === escapeSequence[escapePosition]) {
      escapePosition++;

      if (escapePosition === escapeSequence.length) {
        return i - (escapeSequence.length - 1);
      }
    } else if (escapePosition !== 0) {
      escapePosition = 0;
    }
  }

  return -1;
}

async function* readChunk(response) {
  const escapeSequence = [28, 23, 4];

  const reader = response.body.getReader();
  let { value: chunk, done: readerDone } = await reader.read();
  let part = new Uint8Array(0);
  let endPosition;

  for (;;) {
    if (readerDone) {
      break;
    }

    part = appendChunk(part, chunk);
    endPosition = findIndexEscapeSequence(escapeSequence, part);

    while (endPosition !== -1) {
      yield part.slice(0, endPosition);
      part = part.slice(endPosition + escapeSequence.length);

      endPosition = findIndexEscapeSequence(escapeSequence, part);
    }

    ({ value: chunk, done: readerDone } = await reader.read());
  }
}

function encode(content) {
  const output = [];

  for (let rune of content) {
    output.push(String.fromCharCode(rune));
  }

  return btoa(output.join(''));
}

/**
 * Async image loading
 */
async function fetchThumbnail() {
  let fetchURL = document.location.search;
  if (fetchURL.includes('?')) {
    if (!fetchURL.endsWith('&')) {
      fetchURL += '&';
    }
    fetchURL += 'thumbnail';
  } else {
    fetchURL += '?thumbnail';
  }

  const response = await fetch(fetchURL, { credentials: 'same-origin' });

  if (response.status >= 400) {
    throw new Error('unable to load thumbnails');
  }

  let lazyImageObserver;

  if (typeof lazyLoadThumbnail !== 'undefined' && lazyLoadThumbnail) {
    lazyImageObserver = new IntersectionObserver(async (entries, observer) => {
      for (const entry of entries) {
        if (!entry.isIntersecting) {
          continue;
        }

        const lazyImage = entry.target;
        const parent = lazyImage.parentElement.parentElement;

        const storyThrobber = generateThrobber([
          'throbber-white',
          'throbber-overlay',
        ]);
        parent.appendChild(storyThrobber);

        lazyImage.addEventListener(
          'load',
          () => parent.removeChild(storyThrobber),
          { once: true },
        );

        if (window.webpHero) {
          const response = await fetch(lazyImage.dataset.src, {
            credentials: 'same-origin',
          });
          const content = await response.arrayBuffer();

          const webpMachine = new webpHero.WebpMachine();
          lazyImage.src = await webpMachine.decode(new Uint8Array(content));
          webpMachine.clearCache();
        } else {
          lazyImage.src = lazyImage.dataset.src;
        }

        lazyImageObserver.unobserve(lazyImage);
      }
    });
  }

  for await (let chunk of readChunk(response)) {
    const commaIndex = chunk.findIndex((element) => element === 44);
    if (commaIndex === -1) {
      console.error('invalid line for thumbnail:', line);
      continue;
    }

    const picture = document.getElementById(
      `picture-${String.fromCharCode.apply(null, chunk.slice(0, commaIndex))}`,
    );
    if (!picture) {
      continue;
    }

    const img = new Image();
    img.src = `data:image/webp;base64,${encode(chunk.slice(commaIndex + 1))}`;
    img.alt = picture.dataset.alt;
    img.dataset.src = picture.dataset.src;
    img.classList.add('thumbnail', 'full', 'block');

    replaceContent(picture, img);

    if (lazyImageObserver !== undefined) {
      lazyImageObserver.observe(img);
    }
  }
}

document.addEventListener(
  'readystatechange',
  async (event) => {
    if (event.target.readyState !== 'complete') {
      return;
    }

    if (typeof hasThumbnail === 'undefined' || !hasThumbnail) {
      return;
    }

    let dateTimeFormatter = new Intl.DateTimeFormat(navigator.language, {
      dateStyle: 'full',
      timeStyle: 'long',
    });

    document.querySelectorAll('.date').forEach((item) => {
      item.innerHTML = dateTimeFormatter.format(new Date(item.innerHTML));
    });

    const thumbnailsElem = document.querySelectorAll('[data-thumbnail]');
    if (!thumbnailsElem) {
      return;
    }

    thumbnailsElem.forEach((picture) => {
      replaceContent(picture, generateThrobber(['throbber-white']));
    });

    try {
      await isWebPCompatible();
    } catch (e) {
      await resolveScript(
        'https://unpkg.com/webp-hero@0.0.2/dist-cjs/webp-hero.bundle.js',
        'sha512-DA6h9H5Sqn55/uVn4JI4aSPFnAWoCQYYDXUnvjOAMNVx11///hX4QaFbQt5yWsrIm9hSI5fLJYfRWt3KXneSXQ==',
        'anonymous',
      );
    }

    try {
      await fetchThumbnail();
    } catch (e) {
      console.error(e);
    }

    if (window.webpHero) {
      const webpMachine = new webpHero.WebpMachine();
      webpMachine.polyfillDocument();
      webpMachine.clearCache();
    }
  },
  false,
);
