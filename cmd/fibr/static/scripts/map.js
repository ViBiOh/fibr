async function fetchGeoJSON(geoURL) {
  const response = await fetch(geoURL, { credentials: 'same-origin' });

  if (response.status >= 400) {
    throw new Error('unable to load geojson');
  }

  if (response.status === 204) {
    return null;
  }

  return response.json();
}

async function addStyle(src, integrity, crossorigin) {
  return new Promise((resolve) => {
    const style = document.createElement('link');
    style.rel = 'stylesheet';
    style.href = src;
    style.onload = resolve;

    if (integrity) {
      style.integrity = integrity;
      style.crossOrigin = crossorigin;
    }

    document.querySelector('head').appendChild(style);
  });
}

async function loadLeaflet() {
  const leafletVersion = '1.9.4';

  await addStyle(
    `https://unpkg.com/leaflet@${leafletVersion}/dist/leaflet.css`,
    'sha512-Zcn6bjR/8RZbLEpLIeOwNtzREBAJnUKESxces60Mpoj+2okopSAcSUIUOseddDm0cxnGQzxIR7vJgsLZbdLE3w==',
    'anonymous',
  );
  await addStyle(
    'https://unpkg.com/leaflet.markercluster@1.5.1/dist/MarkerCluster.Default.css',
    'sha512-6ZCLMiYwTeli2rVh3XAPxy3YoR5fVxGdH/pz+KMCzRY2M65Emgkw00Yqmhh8qLGeYQ3LbVZGdmOX9KUjSKr0TA==',
    'anonymous',
  );
  await resolveScript(
    `https://unpkg.com/leaflet@${leafletVersion}/dist/leaflet.js`,
    'sha512-BwHfrr4c9kmRkLw6iXFdzcdWV/PGkVgiIyIWLLlTSXzWQzxuSg4DiQUCpauz/EWjgk5TYQqX/kvn9pG1NpYfqg==',
    'anonymous',
  );
  await resolveScript(
    'https://unpkg.com/leaflet.markercluster@1.5.1/dist/leaflet.markercluster.js',
    'sha512-+Zr0llcuE/Ho6wXRYtlWypMyWSEMxrWJxrYgeAMDRSf1FF46gQ3PAVOVp5RHdxdzikZXuHZ0soHpqRkkPkI3KA==',
    'anonymous',
  );
}

let map;
async function renderMap(geoURL) {
  if (map) {
    map.invalidateSize(); // force re-render
    return;
  }

  const container = document.getElementById('map-container');
  const throbber = generateThrobber(['map-throbber', 'throbber-white']);
  if (container) {
    container.appendChild(throbber);
  }

  await loadLeaflet();

  // create Leaflet map
  map = L.map('map-container', {
    center: [46.227638, 2.213749], // France 🇫🇷
    zoom: 5,
  });

  // add the OpenStreetMap tiles
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution:
      '&copy; <a href="https://openstreetmap.org/copyright">OpenStreetMap contributors</a>',
  }).addTo(map);

  const geojson = await fetchGeoJSON(geoURL);
  if (!geojson || !geojson.features) {
    return;
  }

  const markers = L.markerClusterGroup({
    zoomToBoundsOnClick: false,
  });

  const bounds = [];
  geojson.features.map((f) => {
    const coord = L.GeoJSON.coordsToLatLng(f.geometry.coordinates);

    bounds.push(coord);
    markers.addLayer(
      L.circleMarker(coord).bindPopup(
        `<a href="${f.properties.url}?browser"><img src="${f.properties.url}?thumbnail" alt="Image thumbnail" class="thumbnail-img"></a><br><span>${f.properties.date}</span>`,
        {
          maxWidth: 'auto',
          closeButton: false,
          className: 'thumbnail-popup',
        },
      ),
    );
  });

  markers.on('clusterclick', (a) => {
    map.fitBounds(a.layer.getAllChildMarkers().map((m) => m.getLatLng()));
  });

  // fit bounds of map
  if (bounds.length) {
    map.once('zoomend', () => {
      container.removeChild(throbber);
    });
    map.fitBounds(bounds);
  } else {
    container.removeChild(throbber);
  }

  map.addLayer(markers);
}

document.addEventListener('readystatechange', async (event) => {
  if (event.target.readyState !== 'complete') {
    return;
  }

  if (document.location.hash === '#map') {
    await renderMap(geoURL);
  } else {
    window.addEventListener('popstate', async () => {
      if (document.location.hash === '#map') {
        await renderMap(geoURL);
      }
    });
  }
});
