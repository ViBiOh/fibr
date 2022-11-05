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

// From https://developer.mozilla.org/en-US/docs/Web/API/ReadableStreamDefaultReader/read#example_2_-_handling_text_line_by_line
async function* readLineByLine(response) {
  const utf8Decoder = new TextDecoder('utf-8');
  const reader = response.body.getReader();
  let { value: chunk, done: readerDone } = await reader.read();
  chunk = chunk ? utf8Decoder.decode(chunk, { stream: true }) : '';

  let re = /\r\n|\n|\r/gm;
  let startIndex = 0;

  for (;;) {
    const result = re.exec(chunk);
    if (!result) {
      if (readerDone) {
        break;
      }

      const remainder = chunk.substr(startIndex);
      ({ value: chunk, done: readerDone } = await reader.read());
      chunk =
        remainder + (chunk ? utf8Decoder.decode(chunk, { stream: true }) : '');
      startIndex = re.lastIndex = 0;
      continue;
    }

    yield chunk.substring(startIndex, result.index);
    startIndex = re.lastIndex;
  }

  if (startIndex < chunk.length) {
    yield chunk.substr(startIndex);
  }
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

  for await (let line of readLineByLine(response)) {
    const parts = line.split(',');
    if (parts.length != 2) {
      console.error('invalid line for thumbnail:', line);
      continue;
    }

    const picture = document.getElementById(`picture-${parts[0]}`);
    if (!picture) {
      continue;
    }

    const img = new Image();
    img.src = `data:image/webp;base64,${parts[1]}`;
    img.alt = picture.dataset.alt;
    img.dataset.src = picture.dataset.src;
    img.classList.add('thumbnail', 'full', 'block');

    replaceContent(picture, img);
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
      await fetchThumbnail();
    } catch (e) {
      console.error(e);
    }

    try {
      await isWebPCompatible();
    } catch (e) {
      await resolveScript(
        'https://unpkg.com/webp-hero@0.0.2/dist-cjs/webp-hero.bundle.js',
        'sha512-DA6h9H5Sqn55/uVn4JI4aSPFnAWoCQYYDXUnvjOAMNVx11///hX4QaFbQt5yWsrIm9hSI5fLJYfRWt3KXneSXQ==',
        'anonymous',
      );

      const webpMachine = new webpHero.WebpMachine();
      webpMachine.polyfillDocument();
      webpMachine.clearCache();
    }

    window.dispatchEvent(new Event('thumbnail-done'));
  },
  false,
);

window.addEventListener('thumbnail-done', () => {
  if (typeof lazyLoadThumbnail === 'undefined' || !lazyLoadThumbnail) {
    return;
  }

  const lazyImageObserver = new IntersectionObserver(
    async (entries, observer) => {
      for (const entry of entries) {
        if (!entry.isIntersecting) {
          continue;
        }

        const lazyImage = entry.target;
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
    },
  );

  document
    .querySelectorAll('img.thumbnail')
    .forEach((lazyImage) => lazyImageObserver.observe(lazyImage));
});
