document.addEventListener("readystatechange", (event) => {
  if (event.target.readyState !== "complete") {
    return;
  }

  const pushForm = document.getElementById("push-form");
  const urlInput = document.getElementById("push-url");
  const workerRegister = document.getElementById("worker-register");

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
    const button = pushForm.querySelector("button.bg-primary");
    if (button) {
      button.disabled = true;

      workerRegister.addEventListener("click", async () => {
        const subscription = await registerWorker();

        urlInput.value = subscription.endpoint;

        workerRegister.classList.add("hidden");
        button.disabled = false;
      });
    }
  }
});
