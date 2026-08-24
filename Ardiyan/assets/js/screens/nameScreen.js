// ===== منطق صفحه‌ی ورود نام =====

import { $, showScreen } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { savePlayerData } from '../utils/storage.js';

export function initNameScreen() {
  const input = $('#gameInput');
  const btn = $('#nameConfirmBtn');

  if (!input || !btn) return;

  function confirmName() {
    const name = input.value.trim();
    
    if (!name) {
      showToast('لطفاً نام خود را وارد کنید');
      input.focus();
      return;
    }
    
    if (name.length < 2) {
      showToast('نام باید حداقل ۲ حرف باشد');
      return;
    }
    
    // ذخیره‌سازی نام
    savePlayerData({ playerName: name });
    
    // انیمیشن دکمه
    btn.classList.remove('pulse');
    void btn.offsetWidth;
    btn.classList.add('pulse');
    
    showToast(`درود ${name}! در حال ورود به دنیای آردیان...`);
    
    // رفتن به صفحه‌ی بعد
    setTimeout(() => {
      const badge = document.getElementById('playerNameBadge');
      if (badge) badge.textContent = name;
      showScreen('raceScreen');
    }, 1600);
  }

  btn.addEventListener('click', confirmName);
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') confirmName();
  });

  // اگر نام قبلاً ذخیره شده بود، نمایش در input
  const savedName = localStorage.getItem('ardiyan_playerName');
  if (savedName) {
    input.value = savedName;
  }
}
