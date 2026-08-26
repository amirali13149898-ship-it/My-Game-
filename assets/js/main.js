import { initNameScreen } from './screens/nameScreen.js';
import { initRaceScreen } from './screens/raceScreen.js';
import { initMagicScreen } from './screens/magicScreen.js';
import { showScreen } from './utils/dom.js';
import { getPlayerDataAll } from './utils/storage.js';

console.log('🚀 بازی آردیان در حال راه‌اندازی...');

initNameScreen();
initRaceScreen();
initMagicScreen();

const saved = getPlayerDataAll();
if (saved.name) {
  const badge = document.getElementById('playerNameBadge');
  if (badge) badge.textContent = saved.name;

  if (saved.race && saved.gender && saved.magic) {
    document.getElementById('mapPlayerBadge').textContent = saved.name;
    showScreen('mapScreen');
    import('./screens/mapScreen.js').then(m => m.loadMapForRace(saved.race));
  } else if (saved.race && saved.gender) {
    showScreen('magicScreen');
  } else if (saved.race) {
    showScreen('raceScreen');
  }
}

console.log('✅ بازی آردیان با موفقیت راه‌اندازی شد!');
