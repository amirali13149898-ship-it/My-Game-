// ===== توابع کمکی DOM =====

export function $(selector) {
  return document.querySelector(selector);
}

export function $$(selector) {
  return document.querySelectorAll(selector);
}

export function showScreen(screenId) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  const screen = document.getElementById(screenId);
  if (screen) screen.classList.add('active');
}

export function setBackground(element, imageUrl, fallback = null) {
  const img = new Image();
  img.onload = () => {
    element.style.backgroundImage = `url('${imageUrl}')`;
    element.style.backgroundSize = 'cover';
    element.style.backgroundPosition = 'center';
  };
  img.onerror = () => {
    if (fallback) {
      element.style.backgroundImage = `url('${fallback}')`;
      element.style.backgroundSize = 'cover';
      element.style.backgroundPosition = 'center';
    }
  };
  img.src = imageUrl;
}
