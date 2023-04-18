let fileInput;
let uploadList;
let cancelButton;

/**
 * Noop function for event handler.
 * @param  {Object} e Event
 */
function eventNoop(e) {
  e.preventDefault();
  e.stopPropagation();
}

document.addEventListener('readystatechange', async (event) => {
  if (event.target.readyState !== 'complete') {
    return;
  }

  const dropZone = document.getElementsByTagName('body')[0];

  dropZone.addEventListener('dragover', eventNoop);
  dropZone.addEventListener('dragleave', eventNoop);
  dropZone.addEventListener('drop', (e) => {
    eventNoop(e);

    window.location.hash = '#upload-modal';
    if (fileInput) {
      fileInput.files = e.dataTransfer.files;
      fileInput.dispatchEvent(new Event('change'));
    }
  });
});

/**
 * Convert an ArrayBuffer to a HexString
 * @param  {ArrayBuffer} buffer ArrayBuffer to convert
 * @return {String}      Matching hex string
 */
function bufferToHex(buffer) {
  return Array.prototype.map
    .call(new Uint8Array(buffer), (x) => `00${x.toString(16)}`.slice(-2))
    .join('');
}

/**
 * Compute the sha on input.
 * @param  {Object}          data Data to hash
 * @return {Promise<String>}      Promise that will resolve the sha string
 */
async function sha(data) {
  const buffer = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder('utf-8').encode(data),
  );
  return bufferToHex(buffer);
}

/**
 * Generate file message id.
 * @param  {File} file       File to generate id from.
 * @return {Promise<String>} Promise that will resolve the message id.
 */
async function fileMessageId(file) {
  const hash = await sha(
    JSON.stringify({
      name: file.name,
      size: file.size,
      type: file.type,
      lastModified: file.lastModified,
    }),
  );
  return `upload-file-${hash}`;
}

/**
 * Get human file size.
 * @param  {Number} number Size in bytes
 * @return {String}        Human readable size
 */
function humanFileSize(number) {
  if (number < 1024) {
    return number + 'bytes';
  }

  if (number < 1048576) {
    return (number / 1024).toFixed(0) + ' KB';
  }

  return (number / 1048576).toFixed(0) + ' MB';
}

/**
 * Add upload item content to container.
 * @param  {Element} container Container to append item
 * @param  {File}    file      File to generate an item
 */
async function addUploadItem(container, file) {
  const messageId = await fileMessageId(file);

  const item = document.createElement('div');
  item.id = messageId;
  item.classList.add('flex', 'flex-center', 'margin');

  const itemWrapper = document.createElement('div');
  itemWrapper.classList.add('upload-item');
  item.appendChild(itemWrapper);

  const filename = document.createElement('input');
  filename.id = `${messageId}-filename`;
  filename.classList.add('upload-name', 'full');
  filename.type = 'text';
  filename.value = file.name;
  itemWrapper.appendChild(filename);

  const progressContainer = document.createElement('div');
  progressContainer.classList.add('full', 'flex', 'flex-center');

  const size = document.createElement('em');
  size.innerHTML = humanFileSize(file.size);
  size.style.width = '8rem';
  progressContainer.appendChild(size);

  const progress = document.createElement('progress');
  progress.classList.add('flex-grow', 'margin-left');
  progress.max = 100;
  progress.value = 1;
  progressContainer.appendChild(progress);

  itemWrapper.appendChild(progressContainer);

  const status = document.createElement('span');
  status.classList.add('upload-status');
  item.appendChild(status);

  container.appendChild(item);
}

/**
 * Get list of files to upload.
 */
function getFiles(event) {
  return [].filter
    .call(event.target, (e) => e.nodeName.toLowerCase() === 'input')
    .reduce((acc, cur) => {
      if (cur.type === 'file') {
        acc.files = cur.files;
      } else {
        acc[cur.name] = cur.value;
      }

      return acc;
    }, {});
}

/**
 * Get filename to upload.
 * @param  {String}  messageId Message identifier for file
 * @param  {File}    file      File to upload
 */
function getFilename(messageId, file) {
  const filenameInput = document.getElementById(`${messageId}-filename`);
  if (filenameInput && filenameInput.value) {
    return filenameInput.value;
  }

  return file.name;
}

/**
 * Clear upload status for a file
 * @param {Element} container Container of file status
 */
function clearUploadStatus(container) {
  if (!container) {
    return;
  }

  const statusContainer = container.querySelector('.upload-status');
  if (!statusContainer) {
    return;
  }

  statusContainer.classList.remove('danger');
  statusContainer.classList.remove('success');
}

/**
 * Add upload message for file
 * @param {Element} container Container of file status
 * @param {Element} content   Content of message
 * @param {String}  style     Style of status
 * @param {String}  title     Title of status
 */
async function setUploadStatus(container, content, style, title) {
  if (!container) {
    return;
  }

  const statusContainer = container.querySelector('.upload-status');
  if (!statusContainer) {
    return;
  }

  statusContainer.innerHTML = content;
  statusContainer.classList.add(style);

  if (title) {
    statusContainer.title = title;
  }
}

let uploadFile;
let aborter;

const chunkSize = 1024 * 1024;
let currentUpload = {};

/**
 * Upload file by chunks.
 * @param  {Element} container Container of file status
 * @param  {String}  method    Method for uploading
 * @param  {String}  filename  Name of file uploaded
 * @param  {File}    file      File to upload
 * @return {Promise}           Promise of upload
 */
async function uploadFileByChunks(container, method, filename, file) {
  let progress;
  if (container) {
    progress = container.querySelector('progress');
    clearUploadStatus(container);
  }

  if (file.name !== currentUpload.filename) {
    currentUpload.filename = file.name;
    currentUpload.chunks = [];

    for (let cur = 0; cur < file.size; cur += chunkSize) {
      currentUpload.chunks.push({
        content: file.slice(cur, cur + chunkSize),
        done: false,
      });
    }
  }

  for (let i = 0; i < currentUpload.chunks.length; i++) {
    if (currentUpload.chunks[i].done) {
      continue;
    }

    if (typeof AbortController !== 'undefined') {
      aborter = new AbortController();
    }

    const formData = new FormData();
    formData.append('method', method);
    formData.append('filename', filename);
    formData.append('file', currentUpload.chunks[i].content);

    const response = await fetch('', {
      method: 'POST',
      credentials: 'same-origin',
      signal: aborter.signal,
      headers: {
        'X-Chunk-Upload': true,
        'X-Chunk-Number': i + 1,
        Accept: 'text/plain',
      },
      body: formData,
    });

    if (response.status >= 400) {
      return Promise.reject(await response.text());
    }

    currentUpload.chunks[i].done = true;
    if (progress) {
      progress.value = ((chunkSize * (i + 1)) / file.size) * 100;
    }
  }

  const formData = new FormData();
  formData.append('method', method);
  formData.append('filename', filename);
  formData.append('size', file.size);

  const response = await fetch('', {
    method: 'POST',
    credentials: 'same-origin',
    headers: {
      'X-Chunk-Upload': true,
      Accept: 'text/plain',
    },
    body: formData,
  });

  const output = await response.text();
  if (response.status >= 400) {
    return Promise.reject(output);
  } else {
    currentUpload = {};
    return Promise.resolve(output);
  }
}

/**
 * Upload file with updating progress indicator.
 * @param  {Element} container Container of file status
 * @param  {String}  method    Method for uploading
 * @param  {String}  filename  Name of file uploaded
 * @param  {File}    file      File to upload
 * @return {Promise}           Promise of upload
 */
async function uploadFileByXHR(container, method, filename, file) {
  let progress;
  if (container) {
    progress = container.querySelector('progress');
    clearUploadStatus(container);
  }

  const formData = new FormData();
  formData.append('method', method);
  formData.append('filename', filename);
  formData.append('size', file.size);
  formData.append('file', file);

  return new Promise((resolve, reject) => {
    let xhr = new XMLHttpRequest();
    aborter = xhr;

    if (progress) {
      xhr.upload.addEventListener(
        'progress',
        (e) => (progress.value = parseInt((e.loaded / e.total) * 100, 10)),
        false,
      );
    }

    xhr.addEventListener(
      'readystatechange',
      (e) => {
        if (xhr.readyState === XMLHttpRequest.UNSENT) {
          reject(new Error('request aborted'));
          xhr = undefined;

          return;
        }

        if (xhr.readyState !== XMLHttpRequest.DONE) {
          return;
        }

        if (xhr.status >= 200 && xhr.status < 400) {
          if (progress) {
            progress.value = 100;
          }

          resolve(xhr.responseText);
          xhr = undefined;
        } else {
          reject(e);
          xhr = undefined;
        }
      },
      false,
    );

    xhr.open('POST', '', true);
    xhr.setRequestHeader('Accept', 'text/plain');
    xhr.send(formData);
  });
}

/**
 * Slice FileList from given index.
 * @param  {String} name  Name of the input file element
 * @param  {Number} index Index for the slice
 */
function sliceFileList(name, index) {
  const input = document.getElementById(name);
  const newList = new DataTransfer();

  for (; index < input.files.length; index++) {
    newList.items.add(input.files[index]);
  }

  input.files = newList.files;
}

/**
 * Handle upload submit
 * @param  {Object} event Submit event
 */
async function upload(event) {
  event.preventDefault();

  const uploadButton = document.getElementById('upload-button');
  if (uploadButton) {
    uploadButton.disabled = true;
    replaceContent(uploadButton, generateThrobber(['throbber-white']));
  }

  if (cancelButton) {
    cancelButton.innerHTML = 'Cancel';
  }

  const values = getFiles(event);

  let success = true;
  for (let i = 0; i < values.files.length; i++) {
    const file = values.files[i];

    const messageId = await fileMessageId(file);
    const container = document.getElementById(messageId);

    try {
      const filename = getFilename(messageId, file);

      await uploadFile(container, values.method, filename, file);
      await setUploadStatus(container, '✓', 'success');
    } catch (err) {
      sliceFileList('file', i);
      if (uploadButton) {
        uploadButton.disabled = false;
        uploadButton.innerHTML = 'Retry';
      }
      await setUploadStatus(container, 'X', 'danger', err);

      success = false;
      console.error(err);

      break;
    } finally {
      aborter = undefined;
    }
  }

  if (success) {
    document.location.hash = '#upload-success';
  } else if (cancelButton) {
    cancelButton.innerHTML = 'Close';
  }

  return false;
}

/**
 * Abort current upload.
 */
function abort(e) {
  e.preventDefault();

  if (aborter) {
    aborter.abort();
    aborter = undefined;

    if (cancelButton) {
      cancelButton.innerHTML = 'Close';
    }
  }

  return false;
}

document.addEventListener('readystatechange', async (event) => {
  if (event.target.readyState !== 'complete') {
    return;
  }

  if (typeof chunkUpload !== 'undefined') {
    if (chunkUpload) {
      uploadFile = uploadFileByChunks;
    } else {
      uploadFile = uploadFileByXHR;
    }
  }

  fileInput = document.getElementById('file');
  uploadList = document.getElementById('upload-list');
  cancelButton = document.getElementById('upload-cancel');

  if (fileInput) {
    fileInput.classList.add('opacity');
    fileInput.multiple = true;

    fileInput.addEventListener('change', () => {
      window.location.hash = '#upload-modal';

      replaceContent(uploadList);

      for (const file of fileInput.files) {
        addUploadItem(uploadList, file);
      }
    });

    const uploadButtonLink = document.getElementById('upload-button-link');
    if (uploadButtonLink) {
      uploadButtonLink.addEventListener('click', (e) => {
        eventNoop(e);

        fileInput.click();
      });
    }
  }

  const fileInputLabel = document.getElementById('file-label');
  if (fileInputLabel) {
    fileInputLabel.classList.remove('hidden');
    fileInputLabel.innerHTML = 'Choose files...';
  }

  if (uploadList) {
    uploadList.classList.remove('hidden');
  }

  if (cancelButton) {
    cancelButton.addEventListener('click', goBack);
  }

  const form = document.getElementById('upload-form');
  if (form) {
    form.addEventListener('submit', upload);
  }
});
