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

  // ---------- Reset UI to initial state ----------
  function reset() {
    raceBtns.forEach(btn => btn.classList.remove('selected'));
    genderBtns.forEach(btn => btn.classList.remove('selected'));
    genderBox.style.display = 'none';
    confirmBtn.disabled = true;
    selectedRace = null;
    selectedGender = null;
    container.style.backgroundImage = `url('${CASTLE_BG}')`;
  }

  // ---------- Race selection ----------
  raceBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      // Deselect all races
      raceBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');

      selectedRace = btn.dataset.race;
      setBackground(container, RACE_IMAGES[selectedRace], CASTLE_BG);

      // Show gender selection and reset gender state
      genderBox.style.display = 'flex';
      genderBtns.forEach(b => b.classList.remove('selected'));
      selectedGender = null;
      confirmBtn.disabled = true;
    });
  });

  // ---------- Gender selection ----------
  genderBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      genderBtns.forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      selectedGender = btn.dataset.gender;
      confirmBtn.disabled = false;
    });
  });

  // ---------- Confirm button ----------
  confirmBtn.addEventListener('click', () => {
    if (selectedRace && selectedGender) {
      savePlayerData({ race: selectedRace, gender: selectedGender });
      showToast('نژاد و جنسیت ثبت شد. حالا جادو را انتخاب کن!', 2000);
      setTimeout(() => showScreen('magicScreen'), 1000);
    }
  });

  // Reset everything when screen initializes
  reset();
}

export function resetRaceScreen() {
  const raceBtns = $$('.race-btn');
  const genderBtns = $$('.gender-btn');
  const genderBox = $('#genderBox');
  const confirmBtn = $('#raceConfirmBtn');
  const container = $('#gameContainer');

  raceBtns.forEach(btn => btn.classList.remove('selected'));
  genderBtns.forEach(btn => btn.classList.remove('selected'));

  if (genderBox) genderBox.style.display = 'none';
  if (confirmBtn) confirmBtn.disabled = true;

  selectedRace = null;
  selectedGender = null;

  if (container) {
    container.style.backgroundImage = `url('${CASTLE_BG}')`;
  }
}
