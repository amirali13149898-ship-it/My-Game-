// assets/js/utils/sound.js

let clickSound = null;

function getClickSound() {
  if (!clickSound) {
    clickSound = new Audio('./assets/sounds/click.wav'); // مسیر نسبی مطمئن‌تر
    clickSound.volume = 0.45;
    clickSound.preload = 'auto';
  }
  return clickSound;
}

export function playClick() {
  try {
    const sound = getClickSound();
    sound.currentTime = 0;
    const promise = sound.play();
    if (promise !== undefined) {
      promise.catch(err => {
        console.warn('صدا پخش نشد:', err.message);
      });
    }
  } catch (err) {
    console.warn('خطا در پخش صدا:', err);
  }
}
