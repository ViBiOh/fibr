/**
 * Generate a throbber.
 * @return {Element}
 */
function generateThrobber(classNames = []) {
  const wrapper = document.createElement('div');
  wrapper.classList.add('throbber');
  classNames.forEach((className) => wrapper.classList.add(className));

  for (let i = 1; i < 4; i++) {
    const dot = document.createElement('div');
    dot.classList.add('throbber-dot', `throbber-dot-${i}`);
    wrapper.appendChild(dot);
  }

  return wrapper;
}
