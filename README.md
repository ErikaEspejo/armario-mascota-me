# Armario Mascota - Backend

Backend en Go para sistema de inventario y catálogo con integración de Google Drive.

## Configuración

### Variables de Entorno

Crea un archivo `.env` en la raíz del proyecto (puedes copiar de `.env.example`):

```bash
# Google Drive Credentials
# Opción 1: JSON completo como variable de entorno (recomendado para producción)
GOOGLE_APPLICATION_CREDENTIALS_JSON={"type":"service_account","project_id":"...","private_key":"...","client_email":"..."}

# Opción 2: Path al archivo JSON (para desarrollo local)
# GOOGLE_APPLICATION_CREDENTIALS=secrets/armario-mascota-aeeb428d158d.json

# Google Drive Folder ID
GOOGLE_DRIVE_FOLDER_ID=1TtK0fnadxl3r1-8iYlv2GFf5LgdKxmID

# PostgreSQL Database Connection
# Opción 1: URL completa (recomendado)
DATABASE_URL=postgres://user:password@localhost:5432/armario_mascota?sslmode=disable

# Opción 2: Variables individuales
# DB_HOST=localhost
# DB_PORT=5432
# DB_USER=postgres
# DB_PASSWORD=password
# DB_NAME=armario_mascota
# DB_SSLMODE=disable
```

### Para Desarrollo Local

1. Copia `.env.example` a `.env`:
   ```bash
   cp .env.example .env
   ```

2. Edita `.env` con tus credenciales locales

3. Ejecuta el proyecto:
   ```bash
   go run main.go
   ```

### Para Producción

En producción, configura las variables de entorno directamente en el sistema o contenedor:

```bash
# Opción 1: Credenciales JSON como variable de entorno (recomendado)
export GOOGLE_APPLICATION_CREDENTIALS_JSON='{"type":"service_account",...}'
export GOOGLE_DRIVE_FOLDER_ID=1TtK0fnadxl3r1-8iYlv2GFf5LgdKxmID
export DATABASE_URL=postgres://user:password@host:port/dbname?sslmode=disable
export ENV=production

# Opción 2: Path al archivo JSON (alternativa)
# export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
# export GOOGLE_DRIVE_FOLDER_ID=1TtK0fnadxl3r1-8iYlv2GFf5LgdKxmID
```

## Endpoints

- `GET /ping` - Health check
- `GET /admin/design-assets/sync?folderId=XXXX` - Lista archivos de Google Drive
- `POST /admin/design-assets/sync-db?folderId=XXXX` - Sincroniza archivos de Drive a PostgreSQL

## Estructura del Proyecto

```
armario-mascota-me/
├── app/
│   ├── controller/     # HTTP controllers
│   └── router/         # Route configuration
├── db/                 # Database connection
├── models/             # Data models
├── repository/         # Data access layer
├── service/            # Business logic
└── utils/              # Utility functions
```




