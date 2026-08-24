// ===== فایل ورودی اصلی =====

import { initNameScreen } from './screens/nameScreen.js';
import { initRaceScreen } from './screens/raceScreen.js';
import { initMagicScreen } from './screens/magicScreen.js';
import { showScreen } from './utils/dom.js';
import { getPlayerDataAll } from './utils/storage.js';

// ===== راه‌اندازی همه‌ی صفحات =====
console.log('🚀 بازی آردیان در حال راه‌اندازی...');

// مقداردهی اولیه هر صفحه
initNameScreen();
initRaceScreen();
initMagicScreen();

// ===== بررسی وضعیت ذخیره‌شده =====
const savedData = getPlayerDataAll();

if (savedData.name) {
  // اگر نام ذخیره شده، نمایش در صفحه‌ی مناسب
  const nameBadge = document.getElementById('playerNameBadge');
  if (nameBadge) nameBadge.textContent = savedData.name;
  
  // اگر همه‌ی اطلاعات تکمیل شده، مستقیماً به نقشه برو
  if (savedData.race && savedData.gender && savedData.magic) {
    // همه‌چیز کامل است، مستقیماً به نقشه برو
    const mapBadge = document.getElementById('mapPlayerBadge');
    if (mapBadge) mapBadge.textContent = savedData.name;
    
    showScreen('mapScreen');
    
    // بارگذاری نقشه
    import('./screens/mapScreen.js').then(module => {
      module.loadMapForRace(savedData.race);
    });
  } else if (savedData.race && savedData.gender) {
    // نژاد و جنسیت کامل، برو به صفحه‌ی جادو
    showScreen('magicScreen');
  } else if (savedData.race) {
    // فقط نژاد انتخاب شده، برو به صفحه‌ی جنسیت (همان صفحه‌ی نژاد)
    showScreen('raceScreen');
  }
  // در غیر این صورت در صفحه‌ی نام بمان
}

console.log('✅ بازی آردیان با موفقیت راه‌اندازی شد!');
