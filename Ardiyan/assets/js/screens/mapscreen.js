// ===== منطق صفحه‌ی نقشه =====

import { $ } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { RACE_MAPS, DEFAULT_BG } from '../config.js';

let panzoom = null;
let isInitializing = false;
let currentRace = null;

export function loadMapForRace(race) {
  // اگر نژاد انتخاب نشده، از انسان استفاده کن
  const raceKey = race || localStorage.getItem('ardiyan_race') || 'انسان';
  currentRace = raceKey;
  
  const data = RACE_MAPS[raceKey];
  if (!data) {
    showToast('نقشه‌ای برای این نژاد وجود ندارد!');
    return;
  }

  const img = document.getElementById('map');
  const wrapper = document.getElementById('map-wrapper');
  const container = document.getElementById('map-container');
  const errorDiv = document.getElementById('error-msg');

  if (!img || !wrapper) {
    console.error('المان‌های نقشه پیدا نشدند!');
    return;
  }

  // مخفی کردن خطا
  if (errorDiv) errorDiv.style.display = 'none';

  // حذف نشانگرهای قبلی
  wrapper.querySelectorAll('.city-hit').forEach(el => el.remove());

  // تنظیم تصویر نقشه
  img.src = data.src;
  img.onload = () => {
    initPanzoom(data.cities);
  };
  img.onerror = () => {
    console.warn('نقشه بارگذاری نشد، استفاده از پیش‌فرض');
    img.src = DEFAULT_BG;
    showToast('نقشه بارگذاری نشد، تصویر پیش‌فرض نمایش داده می‌شود.', 3000);
    // سعی می‌کنیم با تصویر پیش‌فرض هم کار کنیم
    setTimeout(() => initPanzoom(data.cities), 500);
  };
}

function initPanzoom(cities) {
  if (isInitializing) return;
  isInitializing = true;

  const img = document.getElementById('map');
  const wrapper = document.getElementById('map-wrapper');
  const container = document.getElementById('map-container');
  const errorDiv = document.getElementById('error-msg');

  if (!img || !wrapper) {
    isInitializing = false;
    return;
  }

  // اگر تصویر هنوز بارگذاری نشده، صبر کن
  if (!img.complete || img.naturalWidth === 0) {
    setTimeout(() => initPanzoom(cities), 300);
    return;
  }

  // مخفی کردن خطا
  if (errorDiv) errorDiv.style.display = 'none';

  // ساخت نشانگرهای شهر
  if (cities && cities.length) {
    cities.forEach(c => {
      const hit = document.createElement('div');
      hit.className = 'city-hit';
      hit.style.left = c.left + '%';
      hit.style.top = c.top + '%';
      hit.dataset.city = c.id;
      
      const icon = document.createElement('div');
      icon.className = 'city-icon';
      hit.appendChild(icon);
      wrapper.appendChild(hit);
    });
  }

  // اگر panzoom قبلاً وجود داشت، آن را نابود کن
  if (panzoom) {
    try { panzoom.destroy(); } catch(e) {}
    panzoom = null;
  }

  // محاسبه scale اولیه برای کاور کامل صفحه
  const coverScale = Math.max(
    window.innerWidth / img.naturalWidth,
    window.innerHeight / img.naturalHeight
  );

  // راه‌اندازی Panzoom
  try {
    panzoom = Panzoom(wrapper, {
      startScale: coverScale,
      minScale: coverScale * 0.5,
      maxScale: Math.max(coverScale * 5, 12),
      contain: 'outside',
      cursor: 'grab',
      touchAction: 'none',
      animate: false,
      duration: 150
    });

    // مرکز کردن نقشه
    const centerX = (img.naturalWidth * coverScale - window.innerWidth) / 2;
    const centerY = (img.naturalHeight * coverScale - window.innerHeight) / 2;
    panzoom.pan(-centerX, -centerY, { animate: false });

    // ذخیره در window برای دسترسی از بیرون
    window.panzoom = panzoom;

    // مدیریت تغییر اندازه‌ی پنجره
    window.removeEventListener('resize', handleResize);
    window.addEventListener('resize', handleResize);

  } catch (e) {
    console.error('خطا در راه‌اندازی Panzoom:', e);
    const errorDiv2 = document.getElementById('error-msg');
    if (errorDiv2) {
      document.getElementById('error-text').textContent = 'خطا در راه‌اندازی نقشه: ' + e.message;
      errorDiv2.style.display = 'block';
    }
  }

  isInitializing = false;
}

function handleResize() {
  if (!panzoom) return;
  
  const img = document.getElementById('map');
  if (!img || !img.naturalWidth) return;

  const coverScale = Math.max(
    window.innerWidth / img.naturalWidth,
    window.innerHeight / img.naturalHeight
  );

  panzoom.zoomTo(coverScale, {
    animate: true,
    duration: 200,
    focal: { x: window.innerWidth / 2, y: window.innerHeight / 2 }
  });
}

// کلیک روی شهرها (با تفویض رویداد)
document.addEventListener('DOMContentLoaded', () => {
  const wrapper = document.getElementById('map-wrapper');
  if (wrapper) {
    wrapper.addEventListener('click', (e) => {
      const hit = e.target.closest('.city-hit');
      if (!hit) return;
      
      const cityId = hit.dataset.city;
      showToast(`وارد شهر شماره ${cityId} شدی!`);
      
      // در نسخه‌ی کامل، اینجا می‌توانید به سرور درخواست بزنید
    });
  }
});

// تابع رفرش برای استفاده در صورت نیاز
export function refreshMap() {
  if (panzoom) {
    try { panzoom.destroy(); } catch(e) {}
    panzoom = null;
  }
  loadMapForRace(currentRace);
}

// کلین‌آپ در صورت خروج از صفحه
export function cleanupMap() {
  if (panzoom) {
    try { panzoom.destroy(); } catch(e) {}
    panzoom = null;
  }
  window.removeEventListener('resize', handleResize);
}
