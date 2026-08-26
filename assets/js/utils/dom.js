export function $(sel) {
  return document.querySelector(sel);
}

export function $$(sel) {
  return document.querySelectorAll(sel);
}

export function showScreen(id) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  const el = document.getElementById(id);
  if (el) el.classList.add('active');
}

export function setBackground(el, url, fallback) {
  const img = new Image();
  img.onload = () => {
    el.style.backgroundImage = `url('${url}')`;
  };
  img.onerror = () => {
    if (fallback) el.style.backgroundImage = `url('${fallback}')`;
  };
  img.src = url;
}
