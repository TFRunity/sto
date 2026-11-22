# Приложение инженера

Android-каркас на React Native и Expo. Подними nginx с Gateway и запусти
Android Emulator. После установки зависимостей (`npm ci --prefix frontend/mobile`)
из корня STO запускай `make mobile`.

Android Emulator обращается к nginx по `http://10.0.2.2/api`. Для физического
устройства задай `EXPO_PUBLIC_API_URL` в `.env.local` с LAN-адресом компьютера
и префиксом `/api`. Для веба `make web` направляет `/api` в локальный nginx
по `http://127.0.0.1`. При другом порте задай `STO_NGINX_ORIGIN` при запуске веба.
