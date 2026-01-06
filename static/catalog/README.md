# Assets del Catálogo

Coloca aquí los archivos estáticos para el catálogo:

## Archivos requeridos:

- **`logo.png`** o **`logo.jpg`** - Logo de la empresa/marca
  - Se mostrará en el encabezado de cada página del catálogo
  - Tamaño recomendado: máximo 150px de ancho x 60px de alto
  - Formatos soportados: PNG, JPG, JPEG

- **`background.png`** o **`background.jpg`** - Imagen de fondo para el catálogo
  - Se usará como fondo de cada página del catálogo
  - Tamaño recomendado: 210mm x 297mm (A4) o patrón repetible
  - Formatos soportados: PNG, JPG, JPEG
  - Opcional: Si no se proporciona, se usará fondo blanco

## Uso:

1. Coloca los archivos en esta carpeta (`static/catalog/`)
2. Los archivos serán servidos automáticamente a través del endpoint `/static/catalog/`
3. Para PDF/PNG, las imágenes se convierten automáticamente a base64
4. Si no colocas los archivos, el catálogo funcionará sin logo ni fondo

## Ejemplo de estructura:

```
static/
  catalog/
    logo.png          ← Logo de la empresa
    background.png    ← Imagen de fondo (opcional)
```

## Notas:

- Los archivos son opcionales. Si no existen, el catálogo se generará sin ellos.
- Para mejor calidad en PDF, usa imágenes de alta resolución.
- El fondo se aplica con opacidad para no interferir con la legibilidad del texto.

