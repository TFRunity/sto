# Приложение инженера

Android-каркас на React Native и Expo. Подними nginx с Gateway и запусти
Android Emulator. После установки зависимостей (`npm ci --prefix frontend/mobile`)
из корня STO запускай `make mobile`.

Android Emulator обращается к nginx по `http://10.0.2.2/api`. Для физического
устройства задай `EXPO_PUBLIC_API_URL` в `.env.local` с LAN-адресом компьютера
и префиксом `/api`. Для веба `make web` направляет `/api` в локальный nginx
по `http://127.0.0.1`. При другом порте задай `STO_NGINX_ORIGIN` при запуске веба.

## Уведомления Android

Для push нужна Android development-сборка на устройстве или эмуляторе с
Google Play services. В Firebase зарегистрируй Android-приложение
`ru.sto.engineer`, а в Expo настрой учётные данные FCM v1. Укажи в
`.env.local` `EXPO_PUBLIC_EAS_PROJECT_ID` и `GOOGLE_SERVICES_FILE`. Для
локальной сборки файл `google-services.json` можно положить в корень
приложения и использовать `GOOGLE_SERVICES_FILE=./google-services.json`.

После входа включи «Получать уведомления» в профиле и выдай разрешение Android.
Серверные сценарии и статус проверки описаны в
[задачах вечера 5](../../stuff/tasks/v3/evening-5.md).
[Настройка Expo Push](https://docs.expo.dev/push-notifications/push-notifications-setup/)
