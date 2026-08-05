# frontend

Стек: React + TypeScript, Vite, MobX, Ant Design 6, Axios. ESLint + Prettier + Stylelint для линтинга/форматирования.

## Структура (Feature-Sliced Design)

```
src/
  app/        # инициализация приложения: провайдеры, глобальные стили, корневой компонент
  pages/      # страницы — композиция виджетов/фич под конкретный роут
  widgets/    # самостоятельные крупные блоки UI
  features/   # пользовательские сценарии, меняющие состояние
  entities/   # бизнес-сущности и их базовое отображение
  shared/     # переиспользуемый код без привязки к бизнес-логике (api, config, lib, ui)
```

Слои доступны по алиасам: `@app`, `@pages`, `@widgets`, `@features`, `@entities`, `@shared` (настроены в `tsconfig.app.json` и `vite.config.ts`).

## Команды

```
npm run dev             # dev-сервер
npm run build           # сборка
npm run preview         # предпросмотр сборки
npm run lint            # ESLint
npm run lint:fix
npm run format          # Prettier
npm run format:check
npm run stylelint       # Stylelint (CSS)
npm run stylelint:fix
```

## Переменные окружения

Скопировать `.env.example` в `.env` и указать `VITE_API_URL` — адрес бэкенда.
