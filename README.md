# telegram-bot

Bot de Telegram para la organización de Telestas. Gestiona una lista global
de tareas (compartida entre el grupo y los chats privados autorizados),
persistida en SQLite.

Escrito en Go, corre por **polling** (sin webhook por ahora).

## Comandos

| Comando | Descripción |
|---|---|
| `/new <texto>` | Crear tarea normal |
| `/newp <texto>` | Crear tarea prioritaria 🔥 (aparece arriba en `/list`) |
| `/list` | Listar tareas abiertas (prioritarias primero) |
| `/edit <id> <texto>` | Modificar el texto de una tarea |
| `/priority <id>` | Alternar prioridad de una tarea |
| `/done <id>` | Cerrar tarea (marcarla como hecha) |
| `/delete <id>` | Borrar tarea |
| `/version` | Versión del bot |
| `/help` | Mostrar la lista de comandos |

## Acceso

- En **grupos / supergrupos** donde esté el bot: cualquier miembro puede usarlo.
- En **chat privado**: solo los IDs de Telegram listados en `ALLOWED_USERS`
  (separados por coma).

En grupos, BotFather pone el bot en *privacy mode* por defecto y el bot solo
ve mensajes que lo mencionan. Si quieres que lea cualquier comando, en
BotFather: `/setprivacy` → `Disable`, y vuelve a añadirlo al grupo.

## Variables de entorno

| Variable | Requerida | Descripción |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | sí | Token del bot que da [@BotFather](https://t.me/BotFather) |
| `ALLOWED_USERS` | no | IDs (CSV) autorizados en chat privado. Tu ID lo da [@userinfobot](https://t.me/userinfobot) |
| `IMAGE` | sí (para compose) | Nombre de la imagen sin tag, p.ej. `usuario/telegram-bot`. La CI lo inyecta en deploy |
| `IMAGE_TAG` | sí (para compose) | Tag de la imagen, p.ej. `latest` o `v0.0.1`. La CI lo inyecta con el tag de git |
| `DB_PATH` | no | Ruta de la BD (default: `tasks.db` en local, `/data/tasks.db` en Docker) |
| `TZ` | no | Zona horaria, default `UTC` |

## Desarrollo local

```bash
cp .env.example .env   # rellena TELEGRAM_BOT_TOKEN y ALLOWED_USERS
docker compose up --build
```

`--build` es necesario la primera vez (o tras cambios) porque el `image:` del
compose apunta al registry; con `--build` Docker lo construye y lo etiqueta
localmente. Para iterar sin Docker:

```bash
export TELEGRAM_BOT_TOKEN=... ALLOWED_USERS=...
go run .
```

## Releases y deploy automático

El workflow [.github/workflows/release.yml](.github/workflows/release.yml) se
dispara al hacer push de un tag con formato `vX.Y.Z` y hace dos cosas:

1. **build** (en `ubuntu-latest`): construye la imagen y la publica en
   Docker Hub como:
   - `<DOCKERHUB_USERNAME>/telegram-bot:<tag>` (ej. `:v0.0.1`)
   - `<DOCKERHUB_USERNAME>/telegram-bot:latest`
2. **deploy** (en el runner self-hosted `tl-havtel-runner`):
   - crea `/opt/telegram-bot` si no existe,
   - copia / sobrescribe `compose.yaml`,
   - `docker compose pull && docker compose up -d` (toma la imagen recién publicada).

### Secretos requeridos en GitHub

- `DOCKERHUB_USERNAME` — usuario de Docker Hub.
- `DOCKERHUB_TOKEN` — access token de Docker Hub (Account Settings → Security).

### Pre-requisitos en el runner `tl-havtel-runner`

- Docker + plugin `compose` instalados, el usuario del runner en el grupo `docker`.
- `sudo` sin password al menos la primera vez (para crear `/opt/telegram-bot`).
- Archivo `/opt/telegram-bot/.env` creado **a mano** con los secretos del bot:
  ```
  TELEGRAM_BOT_TOKEN=123:abc
  ALLOWED_USERS=11111111,22222222
  ```
  El workflow nunca toca este archivo; las variables `IMAGE` e `IMAGE_TAG`
  que necesita `compose.yaml` las inyecta el propio job vía env, no se
  escriben al disco.

### Publicar una versión

```bash
git tag v0.0.1
git push origin v0.0.1
```

Eso es todo: la CI construye, publica y redespliega en cuestión de minutos.
