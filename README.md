# Pengi Med SaaS

Pengi Med SaaS es una plataforma médica integral (Monorepo) diseñada para gestionar clínicas, pacientes y flujos de trabajo administrativos. El proyecto está dividido en múltiples aplicaciones frontend y un backend centralizado utilizando una arquitectura moderna.

## 🏗️ Arquitectura y Tecnologías

Este repositorio es un **Monorepo** que contiene múltiples aplicaciones y servicios.

### Backend (`apps/api`)
- **Lenguaje:** Go (Golang)
- **Framework Web:** Gin
- **Base de Datos:** PostgreSQL
- **Generación de PDFs:** Gotenberg (HTML a PDF)
- **Arquitectura:** Limpia (Handlers, Servicios, Repositorios)

### Frontends
- **Web App (`apps/web`):** Aplicación principal para doctores y pacientes (React + Vite + TypeScript).
- **Backoffice (`apps/backoffice`):** Panel de administración interno (React + Vite + TypeScript).
- **Landing Page (`apps/landing`):** Sitio web público promocional (Astro).
- **Estilos:** TailwindCSS
- **Componentes:** Shadcn UI / Base UI
- **Estado:** Zustand
- **Validaciones form:** Zod + React Hook Form

### Herramientas de Desarrollo
- **Linter & Formatter:** Biome (`biome.json`)
- **Gestión de tareas:** Just (`Justfile`)
- **Contenedores:** Docker & Docker Compose
- **CI/CD:** GitHub Actions

---

## 📁 Estructura del Proyecto

```text
pengi-med-saas/
├── apps/
│   ├── api/          # Backend principal en Go
│   ├── web/          # Frontend principal (Doctores/Pacientes)
│   ├── backoffice/   # Frontend de administración interna
│   └── landing/      # Sitio web público (Astro)
├── deploy/           # Scripts y configuraciones de despliegue
├── .github/          # Flujos de trabajo de CI/CD (GitHub Actions)
├── docker-compose.dev.yaml # Entorno Docker para desarrollo local
├── biome.json        # Configuración global de Linting/Formating
└── Justfile          # Comandos útiles para desarrollo
```

---

## 🚀 Guía de Inicio Rápido (Desarrollo Local)

### Requisitos Previos
1. [Node.js](https://nodejs.org/) (v20 o superior)
2. [Go](https://go.dev/) (v1.21 o superior)
3. [Docker](https://www.docker.com/) y Docker Compose
4. [Just](https://github.com/casey/just) (Opcional, pero recomendado para correr comandos)
5. `pnpm` o `npm` para gestionar paquetes frontend.

### 1. Iniciar Base de Datos y Servicios Locales
El proyecto requiere PostgreSQL y Gotenberg para funcionar localmente.
```bash
docker-compose -f docker-compose.dev.yaml up db gotenberg -d
```

### 2. Configurar y Levantar el Backend (API)
```bash
cd apps/api
# Duplicar archivo de variables de entorno y ajustarlo si es necesario
cp .env.example .env 
go run cmd/api/main.go
```
*La API estará corriendo en `http://localhost:8080`.*

### 3. Levantar los Frontends

**Para la Web App (Clínica):**
```bash
cd apps/web
npm install
npm run dev
```

**Para el Backoffice:**
```bash
cd apps/backoffice
npm install
npm run dev
```

**Para el Landing Page:**
```bash
cd apps/landing
npm install
npm run dev
```

---

## 🛠️ Comandos Útiles

**Validar código con Biome (Lint & Format):**
```bash
# Valida todo el repositorio
npx @biomejs/biome ci . 

# Formatea el código automáticamente
npx @biomejs/biome format --write .
```

**Correr pruebas del Backend:**
```bash
cd apps/api
go test ./...
```

---

## 🔄 CI/CD (Integración y Despliegue Continuo)

El proyecto utiliza **GitHub Actions**. Actualmente está configurado un flujo de CI (`.github/workflows/ci-biome.yml`) que se ejecuta en cada *Pull Request* o *Push* a la rama `main`:
1. Valida el estilo y reglas de código del Frontend usando **Biome**.
2. Descarga dependencias, compila y corre los tests unitarios del Backend en **Go**.

*(Para desplegar a producción se recomienda usar Docker en un VPS para el backend y servicios serverless como Vercel/Netlify para los frontends).*

---

## 📄 Licencia
Privado / Propietario.
