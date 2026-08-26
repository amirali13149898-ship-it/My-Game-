import { $, showScreen } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { savePlayerData } from '../utils/storage.js';
import { playClick } from '../utils/sound.js';

export function initNameScreen() {
  const input = $('#gameInput');
  const btn = $('#nameConfirmBtn');

  if (!input || !btn) {
    console.error('المنت‌های صفحه نام پیدا نشدند');
    return;
  }

  function confirm() {
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

    // پخش صدا
    try {
      playClick();
    } catch (e) {
      console.warn('خطا در پخش صدا:', e);
    }

    // ذخیره نام
    savePlayerData({ playerName: name });

    // انیمیشن دکمه
    btn.classList.remove('pulse');
    void btn.offsetWidth;
    btn.classList.add('pulse');

    // پیام خوش‌آمدگویی
    showToast(`درود ${name}! در حال ورود به دنیای آردیان...`);

    // رفتن به صفحه بعدی
    setTimeout(() => {
      const badge = document.getElementById('playerNameBadge');
      if (badge) badge.textContent = name;
      showScreen('raceScreen');
    }, 1600);
  }

  // رویدادها
  btn.addEventListener('click', confirm);

  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      confirm();
    }
  });
}
