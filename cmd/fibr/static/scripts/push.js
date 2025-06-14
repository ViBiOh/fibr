document.addEventListener("readystatechange", async (event) => {
  if (event.target.readyState !== "complete") {
    return;
  }

  const pushForm = document.getElementById("push-form");
  if (!pushForm) {
    return;
  }

  const submitButton = pushForm.querySelector("button.bg-primary");

  const urlInput = document.getElementById("push-url");
  const pushFormButton = document.getElementById("push-form-button");
  const workerRegister = document.getElementById("worker-register");
  const workerRegisterWrapper = document.getElementById(
    "worker-register-wrapper",
  );

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

  async function checkPush(endpoint) {
    const response = await fetch(
      `?push&endpoint=${encodeURIComponent(endpoint)}`,
      {
        method: "GET",
        credentials: "same-origin",
      },
    );

    if (response.status >= 400) {
      const payload = await response.text();
      throw new Error(`unable to register push: ${payload}`);
    }

    const webhook = await response.json();
    if (webhook.length) {
      // submitButton.innerHTML = "Unsubscribe";
      pushFormButton.querySelector("img").src = "/svg/push?fill=limegreen";
    }
  }

  async function registerWorker() {
    navigator.serviceWorker.register("/service-worker.js", { scope: `./` });

    const registration = await refreshWorker();
    let subscription = await getSubscription(registration);

    if (!subscription) {
      subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidKey),
      });
    }

    registerPush(subscription);

    return subscription;
  }

  function canNotificationBeEnabled() {
    if (!vapidKey) {
      return false;
    }

    if (!("serviceWorker" in navigator)) {
      return false;
    }

    if (/iphone|ipad/i.test(navigator.userAgent)) {
      return window.navigator.standalone === true;
    }

    return true;
  }

  async function getRegistrationWithTimeout() {
    const timeout = new Promise((resolve) => {
      setTimeout(() => {
        resolve(null);
      }, 1000);
    });

    return Promise.race([refreshWorker(), timeout]);
  }

  async function refreshWorker() {
    const registration = await navigator.serviceWorker.ready;
    return await registration.update();
  }

  async function getSubscription(registration) {
    if (registration) {
      return await registration.pushManager.getSubscription();
    }
  }

  function setupSubscription(subscription) {
    urlInput.value = subscription.endpoint;
    submitButton.disabled = false;
  }

  if (canNotificationBeEnabled()) {
    pushFormButton.classList.remove("hidden");
    submitButton.disabled = true;

    const registration = await getRegistrationWithTimeout();
    const subscription = await getSubscription(registration);

    if (subscription && subscription.endpoint) {
      setupSubscription(subscription);
      checkPush(subscription.endpoint);
    } else {
      workerRegisterWrapper.classList.remove("hidden");
      workerRegister.addEventListener("click", async () => {
        setupSubscription(await registerWorker());
        workerRegisterWrapper.classList.add("hidden");
      });
    }
  }
});
