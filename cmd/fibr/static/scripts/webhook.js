document.addEventListener("readystatechange", (event) => {
  if (event.target.readyState !== "complete") {
    return;
  }

  const urlLabel = document.getElementById("webhook-url-label");
  if (!urlLabel) {
    return;
  }

  const urlWrapper = document.getElementById("webhook-url-wrapper");
  const urlWebhook = document.getElementById("webhook-url");
  const telegramChatID = document.getElementById("telegram-chat-id");

  document
    .getElementById("webhook-kind-raw")
    .addEventListener("change", (e) => {
      if (e.target.value === "raw") {
        urlWebhook.placeholder = "https://website.com/fibr";
        urlLabel.innerHTML = "URL";

        urlWrapper.classList.remove("hidden");
        telegramChatID.classList.add("hidden");
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
      }
    });
});
