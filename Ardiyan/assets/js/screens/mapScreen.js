import { $ } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { RACE_MAPS, DEFAULT_BG } from '../config.js';

let panzoom = null, currentRace = null;

export function loadMapForRace(race) {
  const key = race || localStorage.getItem('ardiyan_race') || 'انسان';
  currentRace = key;
  const data = RACE_MAPS[key];
  if (!data) { showToast('نقشه‌ای برای این نژاد وجود ندارد!'); return; }

  const img = document.getElementById('map');
  const wrapper = document.getElementById('map-wrapper');
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
  const img = document.getElementById('map');
  const wrapper = document.getElementById('map-wrapper');
  if (!img || !img.naturalWidth) { setTimeout(() => initPanzoom(cities), 300); return; }

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

  if (panzoom) { try { panzoom.destroy(); } catch(e) {} panzoom = null; }

  const scale = Math.max(window.innerWidth / img.naturalWidth, window.innerHeight / img.naturalHeight);
  panzoom = Panzoom(wrapper, {
    startScale: scale,
    minScale: scale * 0.5,
    maxScale: Math.max(scale * 5, 12),
    contain: 'outside',
    cursor: 'grab',
    touchAction: 'none'
  });

  const cx = (img.naturalWidth * scale - window.innerWidth) / 2;
  const cy = (img.naturalHeight * scale - window.innerHeight) / 2;
  panzoom.pan(-cx, -cy, { animate: false });

  wrapper.addEventListener('click', e => {
    const hit = e.target.closest('.city-hit');
    if (hit) showToast(`وارد شهر شماره ${hit.dataset.city} شدی!`);
  });

  window.addEventListener('resize', () => {
    if (panzoom && img.naturalWidth) {
      const newScale = Math.max(window.innerWidth / img.naturalWidth, window.innerHeight / img.naturalHeight);
      panzoom.zoomTo(newScale, { animate: true, duration: 200 });
    }
  });
}

export function refreshMap() {
  if (panzoom) { try { panzoom.destroy(); } catch(e) {} panzoom = null; }
  loadMapForRace(currentRace);
    }
