// ===== مدیریت توست (پیام‌های اعلان) =====

const toastEl = document.getElementById('toast');
let timer = null;

export function showToast(message, duration = 2800) {
  if (!toastEl) return;
  
  toastEl.textContent = message;
  toastEl.classList.add('show');
  
  clearTimeout(timer);
  timer = setTimeout(() => {
    toastEl.classList.remove('show');
  }, duration);
}

export function hideToast() {
  if (!toastEl) return;
  toastEl.classList.remove('show');
  clearTimeout(timer);
}
