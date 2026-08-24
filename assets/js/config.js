// assets/js/config.js
const CONFIG = {
    // بک‌گراندهای صفحه نژاد
    RACE_BGS: {
        "انسان": "https://i.ibb.co/5xF7vKz/castle-background.jpg",
        "دروف": "https://i.ibb.co/99JGPLqp/Duck-ai-image-2026-08-21-08-56.jpg",
        "نیمه انسان": "https://i.ibb.co/WW78CQy4/03.jpg",
        "نیمه دارگون": "https://i.ibb.co/B2GKjhvk/Dragon.jpg",
        "الف": "https://imglink.cc/cdn/JS4Lj_e7Re.jpg",
        "فرشته": "https://i.ibb.co/4n11Tjdr/05.jpg"
    },

    // بک‌گراندهای صفحه جادو
    MAGIC_BGS: {
        "آتش": "https://i.ibb.co/fg2vPqR/fire-bg.jpg",
        "آب": "https://i.ibb.co/8dK9vL2/water-bg.jpg",
        "باد": "https://i.ibb.co/7mXz9kP/wind-bg.jpg",
        "خاک": "https://i.ibb.co/3vR2pQ8/earth-bg.jpg",
        "نور": "https://i.ibb.co/1cD5nV7/light-bg.jpg",
        "تاریکی": "https://i.ibb.co/9vP6tR4/dark-bg.jpg"
    },

    // نقشه‌های نژاد (با مسیرهای کامل)
    RACE_MAPS: {
        "انسان": {
            src: "https://i.ibb.co/Y4NtgbRZ/map.jpg",
            cities: [
                { id: 1, left: 50,    top: 45 },
                { id: 2, left: 50,    top: 11 },
                { id: 3, left: 50,    top: 85 },
                { id: 4, left: 12.5,  top: 47 },
                { id: 5, left: 87.3,  top: 47 }
            ]
        },
        "دروف": {
            src: "https://i.ibb.co/99JGPLqp/Duck-ai-image-2026-08-21-08-56.jpg",
            cities: [
                { id: 1, left: 50,    top: 41 },
                { id: 2, left: 50.09, top: 9 },
                { id: 3, left: 50.3,  top: 80 },
                { id: 4, left: 14.2,  top: 47.5 },
                { id: 5, left: 83.25, top: 46.1 }
            ]
        },
        "نیمه انسان": {
            src: "https://i.ibb.co/WW78CQy4/03.jpg",
            cities: [
                { id: 1, left: 50,    top: 47 },
                { id: 2, left: 50,    top: 87.5 },
                { id: 3, left: 50.15, top: 10 },
                { id: 4, left: 11.29, top: 48.3 },
                { id: 5, left: 88.95, top: 48.3 }
            ]
        },
        "نیمه دارگون": {
            src: "https://i.ibb.co/B2GKjhvk/Dragon.jpg",
            cities: [
                { id: 1, left: 51.4,  top: 41 },
                { id: 2, left: 50.2,  top: 80.3 },
                { id: 3, left: 47.7,  top: 14 },
                { id: 4, left: 11.5,  top: 51 },
                { id: 5, left: 86.75, top: 50 }
            ]
        },
        "الف": {
            src: "https://imglink.cc/cdn/JS4Lj_e7Re.jpg",
            cities: [
                { id: 1, left: 50,    top: 44 },
                { id: 2, left: 49.49, top: 14 },
                { id: 3, left: 49.99, top: 82.5 },
                { id: 4, left: 12.4,  top: 47.5 },
                { id: 5, left: 88.31, top: 47.5 }
            ]
        },
        "فرشته": {
            src: "https://i.ibb.co/4n11Tjdr/05.jpg",
            cities: [
                { id: 1, left: 50.2,  top: 39 },
                { id: 2, left: 50.1,  top: 79 },
                { id: 3, left: 50.2,  top: 10 },
                { id: 4, left: 11.5,  top: 42 },
                { id: 5, left: 89.45, top: 43 }
            ]
        }
    },

    // مسیرهای پیش‌فرض
    DEFAULT_BG: "https://i.ibb.co/3vR2pQ8/earth-bg.jpg", // یا هر بک‌گراندی که دوست داری
    CASTLE_BG: "https://i.ibb.co/5xF7vKz/castle-background.jpg"
};

export default CONFIG;
