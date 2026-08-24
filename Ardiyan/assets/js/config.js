// ===== تنظیمات ثابت بازی =====

// تصاویر نژادها
export const RACE_IMAGES = {
  "انسان": "assets/images/races/human.jpg",
  "دروف": "assets/images/races/dwarf.jpg",
  "نیمه انسان": "assets/images/races/half-human.jpg",
  "نیمه دارگون": "assets/images/races/half-dragon.jpg",
  "الف": "assets/images/races/elf.jpg",
  "فرشته": "assets/images/races/angel.jpg"
};

// تصاویر پس‌زمینه‌ی جادوها
export const MAGIC_IMAGES = {
  "آتش": "assets/images/magic/fire-bg.jpg",
  "آب": "assets/images/magic/water-bg.jpg",
  "باد": "assets/images/magic/wind-bg.jpg",
  "خاک": "assets/images/magic/earth-bg.jpg",
  "نور": "assets/images/magic/light-bg.jpg",
  "تاریکی": "assets/images/magic/dark-bg.jpg"
};

// اطلاعات نقشه‌ها و شهرها
export const RACE_MAPS = {
  "انسان": {
    src: "assets/images/maps/human-map.jpg",
    cities: [
      { id: 1, left: 50, top: 45 },
      { id: 2, left: 50, top: 11 },
      { id: 3, left: 50, top: 85 },
      { id: 4, left: 12.5, top: 47 },
      { id: 5, left: 87.3, top: 47 }
    ]
  },
  "الف": {
    src: "assets/images/maps/elf-map.jpg",
    cities: [
      { id: 1, left: 50, top: 44 },
      { id: 2, left: 49.49, top: 14 },
      { id: 3, left: 49.99, top: 82.5 },
      { id: 4, left: 12.4, top: 47.5 },
      { id: 5, left: 88.31, top: 47.5 }
    ]
  },
  "دروف": {
    src: "assets/images/maps/dwarf-map.jpg",
    cities: [
      { id: 1, left: 50, top: 41 },
      { id: 2, left: 50.09, top: 9 },
      { id: 3, left: 50.3, top: 80 },
      { id: 4, left: 14.2, top: 47.5 },
      { id: 5, left: 83.25, top: 46.1 }
    ]
  },
  "فرشته": {
    src: "assets/images/maps/angel-map.jpg",
    cities: [
      { id: 1, left: 50.2, top: 39 },
      { id: 2, left: 50.1, top: 79 },
      { id: 3, left: 50.2, top: 10 },
      { id: 4, left: 11.5, top: 42 },
      { id: 5, left: 89.45, top: 43 }
    ]
  },
  "نیمه انسان": {
    src: "assets/images/maps/half-human-map.jpg",
    cities: [
      { id: 1, left: 50, top: 47 },
      { id: 2, left: 50, top: 87.5 },
      { id: 3, left: 50.15, top: 10 },
      { id: 4, left: 11.29, top: 48.3 },
      { id: 5, left: 88.95, top: 48.3 }
    ]
  },
  "نیمه دارگون": {
    src: "assets/images/maps/half-dragon-map.jpg",
    cities: [
      { id: 1, left: 51.4, top: 41 },
      { id: 2, left: 50.2, top: 80.3 },
      { id: 3, left: 47.7, top: 14 },
      { id: 4, left: 11.5, top: 51 },
      { id: 5, left: 86.75, top: 50 }
    ]
  }
};

// تصاویر پس‌زمینه‌ی پیش‌فرض
export const DEFAULT_BG = "assets/images/ui/default-bg.jpg";
export const CASTLE_BG = "assets/images/ui/castle-background.jpg";
