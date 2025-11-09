import * as pdfjs from "https://cdn.jsdelivr.net/npm/pdfjs-dist@5.4.394/+esm";

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "https://cdn.jsdelivr.net/npm/pdfjs-dist@5.4.394/build/pdf.worker.min.mjs",
  import.meta.url,
).toString();

for (const item of document.querySelectorAll("[data-pdf-thumbnail]")) {
  await getPageThumbnail(item.dataset.src, async (blob) => {
    const response = await fetch(`${item.dataset.src}?thumbnail`, {
      method: "POST",
      credentials: "same-origin",
      body: blob,
    });

    if (response.status >= 400) {
      console.error(await response.text());
      return;
    }
  });
}

async function getPageThumbnail(url, onBlob) {
  const response = await fetch(url, {
    credentials: "same-origin",
  });

  if (response.status >= 400) {
    console.error(await response.text());
    return;
  }

  const doc = await pdfjs.getDocument(await response.arrayBuffer()).promise;
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
