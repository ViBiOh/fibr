import * as pdfjs from "https://cdn.jsdelivr.net/npm/pdfjs-dist@4.3.136/+esm";

const thumbnailsElem = document.querySelectorAll("[data-pdf-thumbnail]");
if (thumbnailsElem) {
  pdfjs.GlobalWorkerOptions.workerSrc = new URL(
    "https://cdn.jsdelivr.net/npm/pdfjs-dist@4.3.136/build/pdf.worker.min.mjs",
    import.meta.url,
  ).toString();

  thumbnailsElem.forEach(async (item) => {
    await getPageThumbnail(item.dataset.src, async (blob) => {
      const response = await fetch(`${item.dataset.src}?thumbnail`, {
        method: "POST",
        credentials: "same-origin",
        body: blob,
      });

      const output = await response.text();
      if (response.status >= 400) {
        console.error(output);
      }
    });
  });
}

async function getPageThumbnail(url, onBlob) {
  const doc = await pdfjs.getDocument(url).promise;
  const page = await doc.getPage(1);

  const viewport = page.getViewport({ scale: 1 });
  const canvas = document.createElement("canvas");
  canvas.width = viewport.width;
  canvas.height = viewport.height;

  await page.render({
    canvasContext: canvas.getContext("2d"),
    viewport: viewport,
  }).promise;

  canvas.toBlob(onBlob);
}
