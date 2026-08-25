import { initNameScreen } from './screens/nameScreen.js';
import { initRaceScreen } from './screens/raceScreen.js';
import { initMagicScreen } from './screens/magicScreen.js';
import { showScreen } from './utils/dom.js';

console.log('🚀 بازی آردیان در حال راه‌اندازی...');

// پاک کردن داده‌های قبلی برای تست
localStorage.removeItem('ardiyan_playerName');
localStorage.removeItem('ardiyan_race');
localStorage.removeItem('ardiyan_gender');
localStorage.removeItem('ardiyan_magic');

initNameScreen();
initRaceScreen();
initMagicScreen();

// همیشه از صفحه نام شروع می‌کنیم
showScreen('nameScreen');

console.log('✅ بازی آردیان با موفقیت راه‌اندازی شد!');
