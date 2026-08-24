// ===== منطق صفحه‌ی انتخاب جادو =====

import { $, $$, showScreen, setBackground } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { savePlayerData } from '../utils/storage.js';
import { MAGIC_IMAGES, DEFAULT_BG } from '../config.js';
import { resetRaceScreen } from './raceScreen.js';

let selectedMagic = null;

export function initMagicScreen() {
  const magicBtns = $$('.magic-btn');
  const confirmBtn = $('#magicConfirmBtn');
  const magicInfo = $('#magicInfo');
  const container = $('#magicContainer');
  const backBtn = $('#backToRaceBtn');

  if (!magicBtns.length || !confirmBtn) return;

  function resetMagicSelection() {
    magicBtns.forEach(b => b.classList.remove('selected'));
    selectedMagic = null;
    confirmBtn.disabled = true;
    if (magicInfo) magicInfo.textContent = 'جادوی مورد نظر خود را انتخاب کنید';
    if (container) {
      container.style.backgroundImage = `url('${DEFAULT_BG}')`;
      container.style.backgroundSize = 'cover';
      container.style.backgroundPosition = 'center';
    }
  }

  // انتخاب جادو
  magicBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      magicBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      
      selectedMagic = btn.dataset.magic;
      
      // تغییر پس‌زمینه
      if (container && MAGIC_IMAGES[selectedMagic]) {
        setBackground(container, MAGIC_IMAGES[selectedMagic], DEFAULT_BG);
      }
      
      if (magicInfo) magicInfo.textContent = `جادوی انتخاب‌شده: ${selectedMagic}`;
      confirmBtn.disabled = false;
    });
  });

  // تأیید جادو
  confirmBtn.addEventListener('click', () => {
    if (!selectedMagic) return;
    
    savePlayerData({ magic: selectedMagic });
    showToast(`جادوی ${selectedMagic} انتخاب شد! در حال ورود به نقشه...`, 1800);
    
    // نمایش نام بازیکن در نقشه
    const playerName = localStorage.getItem('ardiyan_playerName') || 'ماجراجو';
    const badge = document.getElementById('mapPlayerBadge');
    if (badge) badge.textContent = playerName;
    
    setTimeout(() => {
      showScreen('mapScreen');
      // راه‌اندازی نقشه با نژاد انتخاب‌شده
      const race = localStorage.getItem('ardiyan_race');
      import('./mapScreen.js').then(module => {
        module.loadMapForRace(race);
      });
    }, 1200);
  });

  // دکمه بازگشت به صفحه‌ی نژاد
  if (backBtn) {
    backBtn.addEventListener('click', () => {
      resetMagicSelection();
      resetRaceScreen(); // ریست کردن صفحه‌ی نژاد
      showScreen('raceScreen');
      showToast('به صفحه انتخاب نژاد برگشتی.', 2000);
    });
  }

  // ریست اولیه
  resetMagicSelection();
}
