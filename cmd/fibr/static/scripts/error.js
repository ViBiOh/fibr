document.addEventListener("readystatechange", (event) => {
  if (event.target.readyState !== "complete") {
    return;
  }

  const link = document.getElementById("go-back");

  if (link) {
    link.setAttribute("href", document.referrer);
    link.addEventListener("click", (e) => {
      e.preventDefault();
      window.addEventListener("popstate", () => {
        window.location.reload(true);
      });
      history.back();
      return false;
    });
  }
});
