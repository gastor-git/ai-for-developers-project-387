# Ошибка opencode GitHub action: `undefined is not an object (evaluating 'p.rest')`

## Симптом
В workflow `.github/workflows/opencode.yml` шаг `anomalyco/opencode/github` падает:

```
Failed to parse JSON
Creating comment...
Error: Unexpected error
undefined is not an object (evaluating 'p.rest')
Error: Process completed with exit code 1.
```

## Причина
Известный баг action'а `anomalyco/opencode/github` (см. upstream-issues anomalyco/opencode #37823, #37831, #37889, не исправлено).

1. GitHub изменил формат OIDC-токена (`sub`) для репозиториев, созданных после 2026-07-15: `repo:octocat@123456/my-repo@789:...` (immutable subject claims).
2. Сервер `api.opencode.ai` неверно парсит `owner/repo` из нового формата → не находит установку GitHub App → отвечает текстовой/HTML 500 вместо JSON.
3. Клиент в `exchangeForAppToken()` вызывает `response.json()` на не-JSON теле → `SyntaxError: Failed to parse JSON`.
4. В `catch` создаётся комментарий через `octoRest.rest...`, но `octoRest` ещё не инициализирован (инициализация идёт после обмена токена) → `undefined is not an object (evaluating 'p.rest')` маскирует реальную ошибку.

## Решение
Отключить сломанный OIDC-обмен и использовать `GITHUB_TOKEN` напрямую:

- в `with:` добавить `use_github_token: true`;
- `permissions.contents: write` (для push и создания PR);
- `checkout` → `persist-credentials: true` (action сам не настраивает git-креды при `use_github_token`);
- `id-token: write` не нужен — удалён.

`GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` уже передаётся в `env` шага (добавлено в коммите `de038c98`), но игнорировалось без input `use_github_token`.

## Требуется после апстрим-фикса
Вернуть `id-token: write` и переключиться обратно на OIDC-поток (или просто перейти на версию action'а с исправлением), когда PR #37831/#37889 выйдут в релиз.
