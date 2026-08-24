const PREFIX = 'ardiyan_';

export function savePlayerData(data) {
  Object.keys(data).forEach(k => localStorage.setItem(PREFIX + k, data[k]));
}

export function getPlayerData(k) { return localStorage.getItem(PREFIX + k); }

export function getPlayerDataAll() {
  return {
    name: localStorage.getItem(PREFIX + 'playerName') || '',
    race: localStorage.getItem(PREFIX + 'race') || null,
    gender: localStorage.getItem(PREFIX + 'gender') || null,
    magic: localStorage.getItem(PREFIX + 'magic') || null
  };
}

export function clearPlayerData() {
  ['playerName','race','gender','magic'].forEach(k => localStorage.removeItem(PREFIX + k));
}
