document.addEventListener('readystatechange', (event) => {
  if (event.target.readyState !== 'complete') {
    return;
  }

  document.querySelectorAll('[data-confirm]').forEach((element) => {
    element.addEventListener('click', (e) => {
      if (
        !confirm(`Are you sure you want to delete ${element.dataset.confirm}?`)
      ) {
        e.preventDefault();
      }
    });
  });
});
