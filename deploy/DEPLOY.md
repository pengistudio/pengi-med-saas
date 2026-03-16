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

### Opción A — Doppler (recomendado)

Doppler sincroniza los secretos al servidor sin tocar archivos en disco.

```bash
# Instalar Doppler CLI
curl -Ls https://cli.doppler.com/install.sh | sh

# Autenticarse (genera un token de servicio en doppler.com)
doppler login

# Configurar el proyecto
doppler setup

# Verificar
doppler secrets
```

Luego para correr docker compose:
```bash
doppler run -- docker compose -f deploy/docker-compose.prod.yaml up -d
```

### Opción B — Archivo .env.prod (más simple)

```bash
cp deploy/.env.prod.example deploy/.env.prod
nano deploy/.env.prod   # completar todos los valores
chmod 600 deploy/.env.prod
```

Generar AUTH_KEY segura:
```bash
openssl rand -base64 32
```

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

# Con Doppler:
doppler run -- docker compose -f docker-compose.prod.yaml up -d --build

# Sin Doppler:
docker compose -f docker-compose.prod.yaml up -d --build
```

Caddy obtiene el certificado SSL automáticamente al primer arranque.
Verificar que todo esté corriendo:

```bash
docker compose -f docker-compose.prod.yaml ps
docker compose -f docker-compose.prod.yaml logs caddy
```

---

## 7. Actualizaciones futuras

```bash
cd ~/pengi-med-saas

# Bajar los últimos cambios
git pull

# Reconstruir y reiniciar solo los servicios afectados
cd deploy
docker compose -f docker-compose.prod.yaml up -d --build api
# o todos:
docker compose -f docker-compose.prod.yaml up -d --build
```

Los contenedores sin cambios no se reinician (Docker detecta el mismo image digest).

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
docker compose -f docker-compose.prod.yaml logs -f

# Solo la API
docker compose -f docker-compose.prod.yaml logs -f api

# Solo Caddy (útil para verificar SSL)
docker compose -f docker-compose.prod.yaml logs -f caddy
```

---

## 10. Comandos útiles

```bash
# Detener todo
docker compose -f docker-compose.prod.yaml down

# Reiniciar un servicio
docker compose -f docker-compose.prod.yaml restart api

# Ver uso de recursos
docker stats

# Limpiar imágenes antiguas (liberar espacio)
docker image prune -f
```

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
