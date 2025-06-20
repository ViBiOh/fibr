document.addEventListener("readystatechange", async (event) => {
  if (event.target.readyState !== "complete") {
    return;
  }

  const pushForm = document.getElementById("push-form-form");
  if (!pushForm) {
    return;
  }

  const submitButton = pushForm.querySelector("button.bg-primary");

  const urlInput = document.getElementById("push-url");
  const pushFormButton = document.getElementById("push-form-button");
  const pushFormMethod = document.getElementById("push-form-method");
  const pushFormID = document.getElementById("push-form-id");
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

  async function checkPush(registration, endpoint) {
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

    const result = await response.json();
    if (result.id) {
      pushFormMethod.value = "DELETE";
      pushFormID.value = result.id;
      submitButton.innerHTML = "Unsubscribe";
      pushFormButton.querySelector("img").src = "/svg/push-ring?fill=limegreen";
    } else if (!result.registered) {
      await registration.unregister();
      enableRegisterButton();
    }
  }

  async function registerWorker() {
    navigator.serviceWorker.register("/service-worker.js");

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
    if (!navigator.serviceWorker.controller) {
      return null;
    }

    const registrationThrobber = generateThrobber([
      "throbber-white",
      "padding",
    ]);

    pushForm.insertBefore(registrationThrobber, workerRegisterWrapper);

    const timeout = new Promise((resolve) => {
      setTimeout(() => {
        resolve(null);
      }, 3000);
    });

    const result = await Promise.race([refreshWorker(), timeout]);

    pushForm.removeChild(registrationThrobber);

    return result;
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

  function enableRegisterButton() {
    workerRegisterWrapper.classList.remove("hidden");
    workerRegister.addEventListener("click", async () => {
      setupSubscription(await registerWorker());
      workerRegisterWrapper.classList.add("hidden");
    });
  }

  if (canNotificationBeEnabled()) {
    pushFormButton.classList.remove("hidden");
    submitButton.disabled = true;

    const registration = await getRegistrationWithTimeout();
    const subscription = await getSubscription(registration);

    if (subscription && subscription.endpoint) {
      setupSubscription(subscription);
      checkPush(registration, subscription.endpoint);
    } else {
      enableRegisterButton();
    }
  }
});
