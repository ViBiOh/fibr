self.addEventListener("install", function (event) {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener("activate", function (event) {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("push", async function (event) {
  const payload = await event.data.json();

  return self.registration.showNotification("FIle BRowser", {
    icon: "/images/favicon/favicon-32x32.png",
    image: payload.image,
    body: payload.description,
    data: {
      url: payload.url,
    },
  });
});

self.addEventListener("notificationclick", function (event) {
  return self.clients.openWindow(event.notification.data.url);
});
