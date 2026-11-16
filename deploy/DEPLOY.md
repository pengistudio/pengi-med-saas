# Guía de Deploy — Pengi Med SaaS

Stack: Docker Compose + Caddy (SSL automático) + DigitalOcean Managed PostgreSQL

---

## Arquitectura en producción

```
Internet
    │
    ▼
Caddy :443 (SSL automático)
    ├── tudominio.com        → landing:80
    ├── app.tudominio.com    → web:80
    ├── admin.tudominio.com  → backoffice:80
    └── api.tudominio.com    → api:8080
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
              RabbitMQ      Gotenberg     sri-xml-signer
                                  │
                    ┌─────────────┘
                    ▼
           DO Managed PostgreSQL
```

---

## 1. Requisitos

### DigitalOcean Droplet
- Ubuntu 22.04 LTS
- Mínimo **2 GB RAM** (el ICD API es el más pesado)
- Recomendado: 4 GB RAM para producción real

### DO Managed PostgreSQL
- Crear en el panel de DigitalOcean → Databases → PostgreSQL 15
- Habilitar la opción **Trusted sources** para permitir solo el Droplet
- Anotar: host, puerto (25060), usuario, contraseña y nombre de la base de datos

### Dominio
- Crear registros DNS tipo **A** apuntando a la IP del Droplet:
  ```
  tudominio.com        → IP_DROPLET
  www.tudominio.com    → IP_DROPLET
  app.tudominio.com    → IP_DROPLET
  admin.tudominio.com  → IP_DROPLET
  api.tudominio.com    → IP_DROPLET
  ```
- Esperar propagación DNS (puede tardar hasta 30 min)

---

## 2. Preparar el Droplet

Conectarse por SSH y ejecutar:

```bash
# Actualizar el sistema
sudo apt update && sudo apt upgrade -y

# Instalar Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker

# Verificar
docker --version
docker compose version
```

---

## 3. Clonar el repositorio

```bash
git clone https://github.com/tu-org/pengi-med-saas.git
cd pengi-med-saas
```

---

## 4. Configurar secretos

Los secretos viven en dos lugares:

- **API (runtime):** `deploy/.env.prod` en el servidor. Compose lo pasa a la
  `api` (`env_file`) y lo usa para interpolar (credenciales de RabbitMQ), por
  eso todos los comandos llevan `--env-file .env.prod`.
- **Frontends (build):** secrets de GitHub (`VITE_API_URL`, `VITE_WEB_APP_URL`,
  `PUBLIC_APP_URL`), que `.github/workflows/deploy.yml` hornea en las imágenes.
  Un build sin ellos falla.

```bash
cp deploy/.env.prod.example deploy/.env.prod
nano deploy/.env.prod   # completar todos los valores
chmod 600 deploy/.env.prod
```

Generar `AUTH_KEY` y las claves de cifrado (`SIGNATURE_ENCRYPTION_KEY`,
`ATTACHMENT_ENCRYPTION_KEY`, `WHATSAPP_ENCRYPTION_KEY`), cada una distinta:
```bash
openssl rand -base64 32
```
Guarda una copia de las claves de cifrado fuera del servidor: si se pierden,
lo que cifran (P12, adjuntos, tokens de WhatsApp) queda ilegible.

La API valida su configuración al arrancar (`apps/api/core/config`): si falta
una variable obligatoria no arranca y el log dice cuál. Cada cambio en
`.env.prod` necesita recrear el contenedor (`up -d api`); `restart` no vuelve a
leer el archivo. Antes de editarlo, guarda una copia:
`cp -p .env.prod .env.prod.bak-$(date +%Y%m%d%H%M)`.

---

## 5. Ajustar el Caddyfile

```bash
nano deploy/Caddyfile
# Reemplazar tudominio.com con tu dominio real en todas las líneas
```

---

## 6. Primer deploy

```bash
cd deploy
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d --build
```

Caddy obtiene el certificado SSL automáticamente al primer arranque.
Verificar que todo esté corriendo:

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yaml ps
docker compose --env-file .env.prod -f docker-compose.prod.yaml logs caddy
```

---

## 7. Actualizaciones futuras

```bash
cd ~/pengi-med-saas

# Bajar los últimos cambios
git pull

# Reconstruir y reiniciar solo los servicios afectados
cd deploy
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d --build api
# o todos:
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d --build
```

Los contenedores sin cambios no se reinician (Docker detecta el mismo image digest).

### Rollback

Cada deploy publica en `ghcr.io/pengistudio/` cada imagen con dos tags: `:latest` y `:<sha completo del commit>` (el `head_sha` del workflow). Los tags por sha se conservan en ghcr.io aunque el droplet haga `docker image prune`.

`docker-compose.prod.yaml` hoy referencia `:latest` literal, así que no basta con exportar una variable. Rollback manual con el compose actual:

```bash
# Traer la imagen vieja y re-etiquetarla como latest localmente
docker pull ghcr.io/pengistudio/pengi-api:<sha>
docker tag ghcr.io/pengistudio/pengi-api:<sha> ghcr.io/pengistudio/pengi-api:latest
cd deploy
# up -d SIN pull (un pull volvería a bajar el latest nuevo)
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d api
```

Repetir para cada servicio afectado (`pengi-web`, `pengi-backoffice`, `pengi-portal`, `pengi-landing`, `pengi-sri-xml-signer`). El siguiente deploy vuelve a `:latest`.

Mejora sugerida (no aplicada): usar `image: ghcr.io/pengistudio/pengi-api:${IMAGE_TAG:-latest}` en el compose; el rollback sería `IMAGE_TAG=<sha> docker compose ... pull && IMAGE_TAG=<sha> docker compose ... up -d`.

---

## 8. Backups

### Base de datos (DO Managed PostgreSQL)
- Activar **Daily Backups** en el panel de DigitalOcean → Databases → tu DB → Backups
- Retención: 7 días incluidos en el plan
- Para backup manual:
  ```bash
  PGPASSWORD=<password> pg_dump -h <DO_DB_HOST> -p 25060 -U <user> pengi_gentoo \
    > backup_$(date +%Y%m%d_%H%M%S).sql
  ```

### Archivos de storage (certificados SRI, PDFs)
El volumen `api_storage` contiene los archivos subidos. Para respaldarlo:

```bash
docker run --rm \
  -v deploy_api_storage:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/storage_$(date +%Y%m%d).tar.gz -C /data .
```

Considerar sincronizar `backups/` a DO Spaces (S3 compatible) con `s3cmd` o `rclone`.

---

## 9. Logs

```bash
# Todos los servicios
docker compose --env-file .env.prod -f docker-compose.prod.yaml logs -f

# Solo la API
docker compose --env-file .env.prod -f docker-compose.prod.yaml logs -f api

# Solo Caddy (útil para verificar SSL)
docker compose --env-file .env.prod -f docker-compose.prod.yaml logs -f caddy
```

---

## 10. Comandos útiles

```bash
# Detener todo
docker compose --env-file .env.prod -f docker-compose.prod.yaml down

# Reiniciar un servicio (no vuelve a leer .env.prod)
docker compose --env-file .env.prod -f docker-compose.prod.yaml restart api

# Aplicar cambios de .env.prod (recrea el contenedor)
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d api

# Ver uso de recursos
docker stats

# Limpiar imágenes antiguas (liberar espacio)
docker image prune -f
```

---

## 11. Monitoreo (Prometheus + Grafana)

`docker-compose.prod.yaml` incluye `prometheus` (scrapea `GET /metrics` de la
API cada 30 s) y `grafana` (dashboard "Pengi API" ya provisionado: requests/s
por ruta, latencia p50/p95/p99, errores 5xx, pool de la DB, goroutines y
memoria). Configuración en `deploy/monitoring/`.

> **Antes de mergear el cambio que los agrega:** añade
> `GRAFANA_ADMIN_PASSWORD` al `.env.prod` del droplet. El compose la exige
> (`${GRAFANA_ADMIN_PASSWORD:?}`): sin ella fallan `pull`/`up -d` de **todo**
> el stack, también el del auto-deploy.

### Activar

En `deploy/.env.prod`:
```bash
METRICS_TOKEN=...            # openssl rand -base64 32 (sin él, /metrics no existe)
GRAFANA_ADMIN_PASSWORD=...   # openssl rand -base64 24
```
Luego (recrea la API para que lea el token):
```bash
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d
```

Cómo llega el token a Prometheus sin commitearlo: compose le pasa **solo**
`METRICS_TOKEN` (no todo `.env.prod`) como variable de entorno; el entrypoint
del contenedor lo escribe en `/tmp/metrics_token` (permisos 600, dentro del
contenedor, fuera del volumen) y `monitoring/prometheus.yml` lo usa con
`authorization.credentials_file`. Si `METRICS_TOKEN` está vacío, Prometheus
arranca igual y el target `api:8000` aparece **down** (la API responde 401 o
404); no se cae nada. Al cambiar el token: `up -d api prometheus`.

El puerto del target (`api:8000`) es el `PORT` de `.env.prod`, el mismo que usa
Caddy; si cambias `PORT`, cambia también `monitoring/prometheus.yml`.

### Abrir Grafana (túnel SSH)

Ninguno de los dos es público: Prometheus no publica puertos (solo la red de
Docker) y Grafana escucha en `127.0.0.1:3000` del droplet. Caddy devuelve 404
a `/metrics` desde Internet. Desde tu máquina:
```bash
ssh -L 3000:localhost:3000 root@<IP_DEL_DROPLET>
```
y abre http://localhost:3000 (usuario `admin`, contraseña
`GRAFANA_ADMIN_PASSWORD`). Registro y acceso anónimo están desactivados.

Para ver Prometheus directamente (targets, consultas):
```bash
docker compose --env-file .env.prod -f docker-compose.prod.yaml exec prometheus \
  wget -qO- http://localhost:9090/api/v1/targets | head -c 600
```

`GRAFANA_ADMIN_PASSWORD` solo se aplica la primera vez (al crear
`grafana.db` en el volumen). Para cambiarla después:
```bash
docker compose --env-file .env.prod -f docker-compose.prod.yaml exec grafana \
  grafana cli admin reset-admin-password '<NUEVA>'
```

### Recursos

| Servicio | RAM máx. | CPU | Disco |
|---|---|---|---|
| prometheus | 384 MB (`mem_limit`) | 0.5 | volumen `prometheus_data`, 15 días o 1 GB (lo que llegue antes) |
| grafana | 256 MB (`mem_limit`) | 0.5 | volumen `grafana_data` |

En la práctica usan bastante menos (un solo target): ~100–150 MB entre los
dos. El tope suma 640 MB de los ~1.9 GB libres del droplet de 4 GB; revisa con
`docker stats`. El dashboard está provisionado desde archivo: los cambios en
la UI no se guardan; edita `monitoring/grafana/dashboards/pengi-api.json`
(y `restart grafana`).

### Desactivar

`stop` es temporal: el siguiente push a `main` hace `up -d` y los vuelve a
levantar.
```bash
docker compose --env-file .env.prod -f docker-compose.prod.yaml stop grafana prometheus
```
Para desactivarlo de verdad, quita (o comenta) los servicios `prometheus` y
`grafana` de `docker-compose.prod.yaml` en un PR y luego, en el droplet:
```bash
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d --remove-orphans
docker volume rm deploy_prometheus_data deploy_grafana_data   # opcional: borra el histórico
```
Quitar `METRICS_TOKEN` (y `up -d api`) además elimina el endpoint `/metrics`.

---

## Estructura final de archivos

```
deploy/
  docker-compose.prod.yaml   ← orquestación producción
  Caddyfile                  ← reverse proxy + SSL
  .env.prod.example          ← template (commiteado)
  .env.prod                  ← secretos reales (ignorado por git)
  .gitignore
  DEPLOY.md                  ← esta guía
  monitoring/
    prometheus.yml           ← scrape de la API (token vía credentials_file)
    grafana/provisioning/    ← datasource Prometheus + proveedor de dashboards
    grafana/dashboards/      ← pengi-api.json

apps/
  api/
    Dockerfile.prod          ← ya existía
  web/
    Dockerfile               ← nuevo
    nginx.conf               ← nuevo
  backoffice/
    Dockerfile               ← nuevo
    nginx.conf               ← nuevo
  landing/
    Dockerfile               ← nuevo
    nginx.conf               ← nuevo
  sri-xml-signer/
    Dockerfile               ← ya existía
```
