# Guía de Despliegue en Render

Este documento describe las variables de entorno necesarias para desplegar el backend en Render usando Docker.

## Variables de Entorno Requeridas

### Base de Datos
- **`DATABASE_URL`** (requerido)
  - URL de conexión a PostgreSQL (Neon)
  - Formato: `postgresql://user:password@host:port/database?sslmode=require`
  - Ejemplo: `postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require`

### URL Base del Servicio
- **`BASE_URL`** (requerido en producción)
  - URL pública completa del servicio en Render
  - Debe incluir el protocolo (https://)
  - Ejemplo: `https://armario-mascota-me.onrender.com`
  - **Importante**: Sin esta variable, el catálogo PDF/PNG fallará porque intentará usar `localhost:8080`

### Google Drive API
- **`GOOGLE_APPLICATION_CREDENTIALS_JSON`** (preferido)
  - Contenido completo del archivo JSON de credenciales de Google Drive API
  - Debe ser el JSON completo como string (incluye saltos de línea)
  - Alternativa: usar `GOOGLE_APPLICATION_CREDENTIALS` con ruta a archivo

- **`GOOGLE_APPLICATION_CREDENTIALS`** (alternativa)
  - Ruta al archivo JSON de credenciales
  - Solo usar si `GOOGLE_APPLICATION_CREDENTIALS_JSON` no está disponible
  - En Docker, el archivo debe estar montado o copiado en el contenedor

### Chrome/Chromium (Opcional)
- **`CHROME_PATH`** (opcional)
  - Ruta al ejecutable de Chrome/Chromium
  - Por defecto, el código detecta automáticamente `/usr/bin/chromium`
  - Solo configurar si Chromium está en una ubicación no estándar
  - Ejemplo: `/usr/bin/chromium-browser`

### Configuración del Servidor
- **`PORT`** (automático)
  - Render inyecta esta variable automáticamente
  - El servidor escucha en `0.0.0.0:$PORT`
  - No es necesario configurarla manualmente

- **`ENV`** (opcional)
  - Configurar como `production` en Render
  - Afecta el comportamiento de fallback de `BASE_URL`
  - Si no está configurado, se asume desarrollo local

### Configuración de Precios (Opcional)
- **`PRICING_CONFIG_PATH`** (opcional)
  - Ruta al archivo de configuración de precios
  - Por defecto: `configs/pricing.json`
  - Solo cambiar si el archivo está en otra ubicación

## Resumen de Variables para Render

Configura estas variables en el dashboard de Render (Environment):

```
DATABASE_URL=postgresql://...
BASE_URL=https://armario-mascota-me.onrender.com
GOOGLE_APPLICATION_CREDENTIALS_JSON={"type":"service_account",...}
ENV=production
```

Opcional:
```
CHROME_PATH=/usr/bin/chromium
PRICING_CONFIG_PATH=configs/pricing.json
```

## Verificación Post-Despliegue

1. Verifica que el servicio responde:
   ```bash
   curl https://armario-mascota-me.onrender.com/ping
   ```

2. Prueba la generación de catálogo:
   ```bash
   curl "https://armario-mascota-me.onrender.com/admin/catalog?size=XS&format=html"
   ```

3. Verifica logs en Render para confirmar:
   - Chrome/Chromium detectado correctamente
   - BASE_URL configurado
   - Conexión a base de datos exitosa

## Troubleshooting

### Error: "exec: google-chrome: executable file not found"
- El Dockerfile instala Chromium automáticamente
- Verifica que el build de Docker se completó correctamente
- Opcionalmente, configura `CHROME_PATH=/usr/bin/chromium`

### Error: "connection refused" al generar PDF/PNG
- Verifica que `BASE_URL` está configurado con la URL pública completa
- No debe incluir trailing slash
- Debe usar `https://` en producción

### Error: "BASE_URL environment variable is not set in production"
- Configura `BASE_URL` en las variables de entorno de Render
- Configura `ENV=production` para activar el modo producción

