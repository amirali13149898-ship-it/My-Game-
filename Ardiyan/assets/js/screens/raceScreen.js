// ===== منطق صفحه‌ی انتخاب نژاد و جنسیت =====

import { $, $$, showScreen, setBackground } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { savePlayerData } from '../utils/storage.js';
import { RACE_IMAGES, CASTLE_BG } from '../config.js';

let selectedRace = null;
let selectedGender = null;

export function initRaceScreen() {
  const raceBtns = $$('.race-btn');
  const genderBtns = $$('.gender-btn');
  const genderBox = $('#genderBox');
  const confirmBtn = $('#raceConfirmBtn');
  const container = $('#gameContainer');

  if (!raceBtns.length || !confirmBtn) return;

  function resetSelection() {
    raceBtns.forEach(b => b.classList.remove('selected'));
    genderBtns.forEach(b => b.classList.remove('selected'));
    if (genderBox) genderBox.style.display = 'none';
    confirmBtn.disabled = true;
    selectedRace = null;
    selectedGender = null;
    if (container) {
      container.style.backgroundImage = `url('${CASTLE_BG}')`;
      container.style.backgroundSize = 'cover';
      container.style.backgroundPosition = 'center';
    }
  }

  // انتخاب نژاد
  raceBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      raceBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      
      selectedRace = btn.dataset.race;
      
      // تغییر پس‌زمینه
      if (container && RACE_IMAGES[selectedRace]) {
        setBackground(container, RACE_IMAGES[selectedRace], CASTLE_BG);
      }
      
      // نمایش دکمه‌های جنسیت
      if (genderBox) genderBox.style.display = 'flex';
      
      // ریست جنسیت
      selectedGender = null;
      genderBtns.forEach(b => b.classList.remove('selected'));
      confirmBtn.disabled = true;
    });
  });

  // انتخاب جنسیت
  genderBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      genderBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      
      selectedGender = btn.dataset.gender;
      confirmBtn.disabled = false;
    });
  });

  // تأیید نهایی
  confirmBtn.addEventListener('click', () => {
    if (selectedRace && selectedGender) {
      savePlayerData({
        race: selectedRace,
        gender: selectedGender
      });
      
      showToast(`نژاد ${selectedRace} و جنسیت ${selectedGender} ثبت شد. حالا جادو را انتخاب کن!`, 2000);
      
      setTimeout(() => {
        showScreen('magicScreen');
        // صفحه‌ی جادو در main.js مقداردهی شده است
      }, 1000);
    }
  });

  // ریست اولیه
  resetSelection();
}

// تابع ریست برای استفاده از بیرون (مثلاً وقتی از جادو برمی‌گردیم)
export function resetRaceScreen() {
  const raceBtns = $$('.race-btn');
  const genderBtns = $$('.gender-btn');
  const genderBox = $('#genderBox');
  const confirmBtn = $('#raceConfirmBtn');
  const container = $('#gameContainer');

  raceBtns.forEach(b => b.classList.remove('selected'));
  genderBtns.forEach(b => b.classList.remove('selected'));
  if (genderBox) genderBox.style.display = 'none';
  if (confirmBtn) confirmBtn.disabled = true;
  selectedRace = null;
  selectedGender = null;
  if (container) {
    container.style.backgroundImage = `url('${CASTLE_BG}')`;
    container.style.backgroundSize = 'cover';
    container.style.backgroundPosition = 'center';
  }
}
