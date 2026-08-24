// ===== مدیریت ذخیره‌سازی در localStorage =====

const PREFIX = 'ardiyan_';

export function savePlayerData(data) {
  Object.keys(data).forEach(key => {
    localStorage.setItem(`${PREFIX}${key}`, data[key]);
  });
}

export function getPlayerData(key) {
  return localStorage.getItem(`${PREFIX}${key}`);
}

export function getPlayerDataAll() {
  return {
    name: localStorage.getItem(`${PREFIX}playerName`) || '',
    race: localStorage.getItem(`${PREFIX}race`) || null,
    gender: localStorage.getItem(`${PREFIX}gender`) || null,
    magic: localStorage.getItem(`${PREFIX}magic`) || null
  };
}

export function clearPlayerData() {
  localStorage.removeItem(`${PREFIX}playerName`);
  localStorage.removeItem(`${PREFIX}race`);
  localStorage.removeItem(`${PREFIX}gender`);
  localStorage.removeItem(`${PREFIX}magic`);
}

export function saveBackup(data) {
  localStorage.setItem(`${PREFIX}backup`, JSON.stringify(data));
}

export function getBackup() {
  const backup = localStorage.getItem(`${PREFIX}backup`);
  return backup ? JSON.parse(backup) : null;
}
