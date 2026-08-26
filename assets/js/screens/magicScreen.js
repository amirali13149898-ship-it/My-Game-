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

  function reset() {
    magicBtns.forEach(b => b.classList.remove('selected'));
    selectedMagic = null;
    confirmBtn.disabled = true;
    if (magicInfo) magicInfo.textContent = 'جادوی مورد نظر خود را انتخاب کنید';
    if (container) container.style.backgroundImage = `url('${DEFAULT_BG}')`;
  }

  magicBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      magicBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      selectedMagic = btn.dataset.magic;
      setBackground(container, MAGIC_IMAGES[selectedMagic], DEFAULT_BG);
      magicInfo.textContent = `جادوی انتخاب‌شده: ${selectedMagic}`;
      confirmBtn.disabled = false;
    });
  });

  confirmBtn.addEventListener('click', () => {
    if (!selectedMagic) return;
    savePlayerData({ magic: selectedMagic });
    showToast(`جادوی ${selectedMagic} انتخاب شد! در حال ورود به نقشه...`, 1800);
    const name = localStorage.getItem('ardiyan_playerName') || 'ماجراجو';
    document.getElementById('mapPlayerBadge').textContent = name;
    setTimeout(() => {
      showScreen('mapScreen');
      const race = localStorage.getItem('ardiyan_race');
      import('./mapScreen.js').then(m => m.loadMapForRace(race));
    }, 1200);
  });

  if (backBtn) {
    backBtn.addEventListener('click', () => {
      reset();
      resetRaceScreen();
      showScreen('raceScreen');
      showToast('به صفحه انتخاب نژاد برگشتی.', 2000);
    });
  }

  reset();
}
