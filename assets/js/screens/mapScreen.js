import { $ } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { RACE_MAPS, DEFAULT_BG } from '../config.js';

let panzoom = null;
let currentRace = null;

export function loadMapForRace(race) {
  const key = race || localStorage.getItem('ardiyan_race') || 'انسان';
  currentRace = key;
  const data = RACE_MAPS[key];
  if (!data) {
    showToast('نقشه‌ای برای این نژاد وجود ندارد!');
    return;
  }

  const img = $('#map');
  const wrapper = $('#map-wrapper');
  wrapper.querySelectorAll('.city-hit').forEach(el => el.remove());

  img.src = data.src;

  img.onload = () => initPanzoom(data.cities);
  img.onerror = () => {
    img.src = DEFAULT_BG;
    showToast('نقشه بارگذاری نشد، تصویر پیش‌فرض نمایش داده می‌شود.', 3000);
    setTimeout(() => initPanzoom(data.cities), 500);
  };
}

function initPanzoom(cities) {
  const img = $('#map');
  const wrapper = $('#map-wrapper');

  // ساخت نشانه‌های شهر
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

  // پاکسازی قبلی
  if (panzoom) {
    try { panzoom.destroy(); } catch(e) {}
    panzoom = null;
  }

  const naturalW = img.naturalWidth;
  const naturalH = img.naturalHeight;

  if (!naturalW || !naturalH) {
    setTimeout(() => initPanzoom(cities), 300);
    return;
  }

  const coverScale = Math.max(window.innerWidth / naturalW, window.innerHeight / naturalH);
  const minScale = coverScale * 0.5;
  const maxScale = Math.max(coverScale * 5, 12);

  // === اینجا همه چیز درست شده ===
  panzoom = Panzoom(wrapper, {
    startScale: coverScale,
    minScale: minScale,
    maxScale: maxScale,
    contain: 'outside',
    cursor: 'grab',
    touchAction: 'auto',      // <--- این خط کلیدی بود (قبلاً 'none' بود)
    direction: 'ltr',         // <--- جهت لمس صحیح (چپ به راست)
    panOnlyWhenZooming: false,
    animate: false
  });

  // موقعیت اولیه مرکز نقشه
  const cx = (naturalW * coverScale - window.innerWidth) / 2;
  const cy = (naturalH * coverScale - window.innerHeight) / 2;
  panzoom.pan(-cx, -cy, { animate: false });

  // هندل resize
  window.addEventListener('resize', () => {
    if (!panzoom || !img.naturalWidth) return;
    const newCover = Math.max(window.innerWidth / img.naturalWidth, window.innerHeight / img.naturalHeight);
    if (Math.abs(newCover - coverScale) < 0.015) return;

    const currentScale = panzoom.getScale();
    const ratio = currentScale / coverScale;
    const newScale = Math.min(Math.max(ratio * newCover, minScale), maxScale);

    panzoom.setOptions({ minScale, maxScale });
    panzoom.zoomTo(newScale, { animate: true, duration: 200 });
  });

  // کلیک شهر
  wrapper.addEventListener('click', e => {
    const hit = e.target.closest('.city-hit');
    if (hit) showToast(`وارد شهر شماره ${hit.dataset.city} شدی!`);
  });
}

export function refreshMap() {
  if (panzoom) {
    try { panzoom.destroy(); } catch(e) {}
    panzoom = null;
  }
  loadMapForRace(currentRace);
      }
