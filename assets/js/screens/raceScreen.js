import { $, \[ , showScreen, setBackground } from '../utils/dom.js';
import { showToast } from '../utils/toast.js';
import { savePlayerData } from '../utils/storage.js';
import { RACE_IMAGES, CASTLE_BG } from '../config.js';
import { playClick } from '../utils/sound.js';

let selectedRace = null;
let selectedGender = null;

export function initRaceScreen() {
  const raceBtns = \]('.race-btn');
  const genderBtns = \[ ('.gender-btn');
  const genderBox = $('#genderBox');
  const confirmBtn = $('#raceConfirmBtn');
  const container = $('#gameContainer');

  function reset() {
    raceBtns.forEach(b => b.classList.remove('selected'));
    genderBtns.forEach(b => b.classList.remove('selected'));
    genderBox.style.display = 'none';
    confirmBtn.disabled = true;
    selectedRace = selectedGender = null;
    container.style.backgroundImage = `url('${CASTLE_BG}')`;
  }

  raceBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      playClick();
      raceBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      selectedRace = btn.dataset.race;
      setBackground(container, RACE_IMAGES[selectedRace], CASTLE_BG);
      genderBox.style.display = 'flex';
      genderBtns.forEach(b => b.classList.remove('selected'));
      selectedGender = null;
      confirmBtn.disabled = true;
    });
  });

  genderBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      playClick();
      genderBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      selectedGender = btn.dataset.gender;
      confirmBtn.disabled = false;
    });
  });

  confirmBtn.addEventListener('click', () => {
    if (selectedRace && selectedGender) {
      playClick();
      savePlayerData({ race: selectedRace, gender: selectedGender });
      showToast(`نژاد و جنسیت ثبت شد. حالا جادو را انتخاب کن!`, 2000);
      setTimeout(() => showScreen('magicScreen'), 1000);
    }
  });

  reset();
}

export function resetRaceScreen() {
  const raceBtns = \]('.race-btn');
  const genderBtns = \[ ('.gender-btn');
  const genderBox = $('#genderBox');
  const confirmBtn = $('#raceConfirmBtn');
  const container = $('#gameContainer');

  raceBtns.forEach(b => b.classList.remove('selected'));
  genderBtns.forEach(b => b.classList.remove('selected'));
  if (genderBox) genderBox.style.display = 'none';
  if (confirmBtn) confirmBtn.disabled = true;
  selectedRace = selectedGender = null;
  if (container) container.style.backgroundImage = `url('${CASTLE_BG}')`;
}
