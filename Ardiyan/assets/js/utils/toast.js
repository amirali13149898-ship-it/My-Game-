const toast = document.getElementById('toast');
let timer;

export function showToast(msg, duration = 2800) {
  toast.textContent = msg;
  toast.classList.add('show');
  clearTimeout(timer);
  timer = setTimeout(() => toast.classList.remove('show'), duration);
}

export function hideToast() {
  toast.classList.remove('show');
  clearTimeout(timer);
}
