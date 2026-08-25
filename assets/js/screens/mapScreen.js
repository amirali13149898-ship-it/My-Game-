import { showToast } from '../utils/toast.js';
import { RACE_MAPS, DEFAULT_BG } from '../config.js';

let panzoom = null;
let currentRace = null;
let coverScale = 1;
let isUserZooming = false;

export function loadMapForRace(race) {
  const key = race || localStorage.getItem('ardiyan_race') || 'انسان';
  currentRace = key;
  const data = RACE_MAPS[key];

  if (!data) {
    showToast('نقشه‌ای برای این نژاد وجود ندارد!');
    return;
  }

  const img = document.getElementById('map');
  const wrapper = document.getElementById('map-wrapper');

  // پاک کردن نشانگرهای قبلی
  wrapper.querySelectorAll('.city-hit').forEach(el => el.remove());

  img.onload = null;
  img.onerror = null;

  img.src = data.src;

  img.onload = () => {
    setTimeout(() => initPanzoom(data.cities), 50);
  };

  img.onerror = () => {
    showToast('نقشه بارگذاری نشد', 2500);
    img.src = DEFAULT_BG;
    setTimeout(() => initPanzoom(data.cities), 400);
  };
}

function initPanzoom(cities) {
  const img = document.getElementById('map');
  const wrapper = document.getElementById('map-wrapper');
  const container = document.getElementById('map-container');

  if (!img || !img.naturalWidth || !img.naturalHeight) {
    setTimeout(() => initPanzoom(cities), 200);
    return;
  }

  // ساخت نشانگر شهرها
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

  // نابود کردن نمونه قبلی
  if (panzoom) {
    try { panzoom.destroy(); } catch (e) {}
    panzoom = null;
  }

  // خیلی مهم: اندازه واقعی عکس رو به wrapper بده
  wrapper.style.width = img.naturalWidth + 'px';
  wrapper.style.height = img.naturalHeight + 'px';

  coverScale = Math.max(
    window.innerWidth / img.naturalWidth,
    window.innerHeight / img.naturalHeight
  );

  if (coverScale < 0.1) coverScale = 1;

  const minScale = coverScale * 0.45;
  const maxScale = Math.max(coverScale * 6, 14);

  panzoom = Panzoom(wrapper, {
    startScale: coverScale,
    minScale: minScale,
    maxScale: maxScale,
    contain: 'outside',
    cursor: 'grab',
    touchAction: 'none',
    animate: false,
    duration: 120,
    // این گزینه‌ها کمک می‌کنن حرکت روان‌تر باشه
    handleStartEvent: (e) => {
      e.preventDefault();
    }
  });

  // مرکز کردن نقشه
  const centerX = (img.naturalWidth * coverScale - window.innerWidth) / 2;
  const centerY = (img.naturalHeight * coverScale - window.innerHeight) / 2;
  panzoom.pan(-centerX, -centerY, { animate: false });

  // جلوگیری از تداخل با اسکرول مرورگر
  container.addEventListener('touchmove', (e) => {
    if (e.target.closest('#map-wrapper')) {
      e.preventDefault();
    }
  }, { passive: false });

  // کلیک روی شهر
  wrapper.onclick = (e) => {
    const hit = e.target.closest('.city-hit');
    if (hit) {
      const cityId = hit.dataset.city;
      showToast(`وارد شهر شماره ${cityId} شدی!`);
    }
  };

  // تغییر سایز صفحه
  window.addEventListener('resize', () => {
    if (!panzoom || !img.naturalWidth || isUserZooming) return;

    const newCover = Math.max(
      window.innerWidth / img.naturalWidth,
      window.innerHeight / img.naturalHeight
    );

    if (Math.abs(newCover - coverScale) < 0.02) return;

    const currentScale = panzoom.getScale();
    const ratio = currentScale / coverScale;
    coverScale = newCover;

    const newMin = coverScale * 0.45;
    const newMax = Math.max(coverScale * 6, 14);
    const newScale = Math.min(Math.max(ratio * coverScale, newMin), newMax);

    panzoom.setOptions({ minScale: newMin, maxScale: newMax });
    panzoom.zoom(newScale, { animate: true });
  });
}

export function refreshMap() {
  if (panzoom) {
    try { panzoom.destroy(); } catch (e) {}
    panzoom = null;
  }
  loadMapForRace(currentRace);
}
