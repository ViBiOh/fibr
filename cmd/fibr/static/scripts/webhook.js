document.addEventListener("readystatechange", (event) => {
  if (event.target.readyState !== "complete") {
    return;
  }

  const urlLabel = document.getElementById("webhook-url-label");
  if (!urlLabel) {
    return;
  }

  const urlInput = document.getElementById("webhook-url");
  const urlWrapper = document.getElementById("webhook-url-wrapper");

  const pushWrapper = document.getElementById("webhook-push-wrapper");
  const urlWebhook = document.getElementById("webhook-url");
  const telegramChatID = document.getElementById("telegram-chat-id");

  const serviceWorkerRegister = document.getElementById(
    "service-worker-register",
  );

  document
    .getElementById("webhook-kind-raw")
    .addEventListener("change", (e) => {
      if (e.target.value === "raw") {
        urlWebhook.placeholder = "https://website.com/fibr";
        urlLabel.innerHTML = "URL";

        urlWrapper.classList.remove("hidden");
        telegramChatID.classList.add("hidden");
        pushWrapper.classList.add("hidden");
      }
    });

  document
    .getElementById("webhook-kind-discord")
    .addEventListener("change", (e) => {
      if (e.target.value === "discord") {
        urlWebhook.placeholder = "https://discord.com/api/webhooks/...";
        urlLabel.innerHTML = "URL";

        urlWrapper.classList.remove("hidden");
        telegramChatID.classList.add("hidden");
        pushWrapper.classList.add("hidden");
      }
    });

  document
    .getElementById("webhook-kind-slack")
    .addEventListener("change", (e) => {
      if (e.target.value === "slack") {
        urlWebhook.placeholder = "https://hooks.slack.com/services/...";
        urlLabel.innerHTML = "URL";

        urlWrapper.classList.remove("hidden");
        telegramChatID.classList.add("hidden");
        pushWrapper.classList.add("hidden");
      }
    });

  document
    .getElementById("webhook-kind-telegram")
    .addEventListener("change", (e) => {
      if (e.target.value === "telegram") {
        urlLabel.innerHTML = "Token";
        urlWebhook.placeholder = "Bot token";

        urlWrapper.classList.remove("hidden");
        telegramChatID.classList.remove("hidden");
        pushWrapper.classList.remove("hidden");
      }
    });

  document
    .getElementById("webhook-kind-push")
    .addEventListener("change", (e) => {
      if (e.target.value === "push") {
        pushWrapper.classList.remove("hidden");
        urlWrapper.classList.add("hidden");
        telegramChatID.classList.add("hidden");
      }
    });

  function generateKey(keyName, subscription) {
    const rawKey = subscription.getKey ? subscription.getKey(keyName) : "";
    return rawKey
      ? btoa(String.fromCharCode.apply(null, new Uint8Array(rawKey)))
      : "";
  }

  function urlBase64ToUint8Array(base64String) {
    var padding = "=".repeat((4 - (base64String.length % 4)) % 4);
    var base64 = (base64String + padding)
      .replace(/\-/g, "+")
      .replace(/_/g, "/");

    var rawData = window.atob(base64);
    var outputArray = new Uint8Array(rawData.length);

    for (var i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }

    return outputArray;
  }

  async function generatePublicKey(subscription) {
    return generateKey("p256dh", subscription);
  }

  async function generateAuthKey(subscription) {
    return generateKey("auth", subscription);
  }

  async function registerPush(subscription) {
    const publicKey = await generatePublicKey(subscription);
    const authKey = await generateAuthKey(subscription);

    const response = await fetch("?push", {
      method: "POST",
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        endpoint: subscription.endpoint,
        publicKey: publicKey,
        auth: authKey,
      }),
    });

    if (response.status >= 400) {
      const payload = await response.text();
      throw new Error(`unable to register push: ${payload}`);
    }
  }

  async function registerWorker() {
    navigator.serviceWorker.register("/service-worker.js", {
      scope: "/",
    });

    let registration = await navigator.serviceWorker.ready;
    registration = await registration.update();

    let subscription = await registration.pushManager.getSubscription();
    if (!subscription) {
      subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidKey),
      });
    }

    registerPush(subscription);

    return subscription;
  }

  if ("serviceWorker" in navigator) {
    document.getElementById("push-webhook-selector").classList.remove("hidden");

    serviceWorkerRegister.addEventListener("click", async () => {
      const subscription = await registerWorker();

      pushWrapper.classList.add("hidden");

      urlInput.value = subscription.endpoint;
    });
  }
});
