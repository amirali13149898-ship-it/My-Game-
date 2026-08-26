import { $, showScreen } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { savePlayerData } from '../utils/storage.js';

export function initNameScreen() {
  const input = $('#gameInput');
  const btn = $('#nameConfirmBtn');

  function confirm() {
    const name = input.value.trim();
    if (!name) { showToast('لطفاً نام خود را وارد کنید'); input.focus(); return; }
    if (name.length < 2) { showToast('نام باید حداقل ۲ حرف باشد'); return; }
    
    savePlayerData({ playerName: name });
    btn.classList.remove('pulse');
    void btn.offsetWidth;
    btn.classList.add('pulse');
    showToast(`درود ${name}! در حال ورود به دنیای آردیان...`);
    
    setTimeout(() => {
      const badge = document.getElementById('playerNameBadge');
      if (badge) badge.textContent = name;
      showScreen('raceScreen');
    }, 1600);
  }

  btn.addEventListener('click', confirm);
  input.addEventListener('keydown', e => { if (e.key === 'Enter') confirm(); });

  const saved = localStorage.getItem('ardiyan_playerName');
  if (saved) input.value = saved;
}
