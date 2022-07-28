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
  fetchURL = document.location.search;
  if (fetchURL.includes('?')) {
    fetchURL += '&thumbnail';
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

window.addEventListener(
  'load',
  async () => {
    const thumbnailsElem = document.querySelectorAll('[data-thumbnail]');
    if (!thumbnailsElem) {
      return;
    }

    try {
      await isWebPCompatible();
    } catch (e) {
      console.error('Your browser is not compatible with WebP format.', e);
      thumbnailsElem.forEach(displayNoThumbnail);
      return;
    }

    thumbnailsElem.forEach((picture) => {
      replaceContent(picture, generateThrobber(['throbber-white']));
    });

    try {
      await fetchThumbnail();
      window.dispatchEvent(new Event('thumbnail-done'));
    } catch (e) {
      console.error(e);
    }
  },
  false,
);
