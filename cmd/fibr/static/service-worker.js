self.addEventListener("install", function (event) {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener("activate", function (event) {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("push", function (event) {
  event.waitUntil(
    self.clients.matchAll().then(function (clientList) {
      console.log(clientList);

      var focused = clientList.some(function (client) {
        return client.focused;
      });

      var notificationMessage;
      if (focused) {
        notificationMessage = "You're still here, thanks!";
      } else if (clientList.length > 0) {
        notificationMessage =
          "You haven't closed the page, " + "click here to focus it!";
      } else {
        notificationMessage =
          "You have closed the page, " + "click here to re-open it!";
      }

      return self.registration.showNotification("FIle BrowseR", {
        body: notificationMessage,
      });
    }),
  );
});

self.addEventListener("notificationclick", function (event) {
  event.waitUntil(
    self.clients.matchAll().then(function (clientList) {
      console.log(clientList);

      if (clientList.length > 0) {
        return clientList[0].focus();
      }

      return self.clients.openWindow("https://ketchup.vibioh.fr");
    }),
  );
});
