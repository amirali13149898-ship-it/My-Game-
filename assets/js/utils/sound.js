const clickSound = new Audio('assets/sounds/click.wav');
clickSound.volume = 0.45;

export function playClick() {
  clickSound.currentTime = 0;
  clickSound.play().catch(() => {});
}
