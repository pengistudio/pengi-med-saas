# Pengi Med SaaS — Development Documentation

Documentación de desarrollo de Pengi Med SaaS, válida para cualquier editor.
Las convenciones generales del proyecto están en [`../CLAUDE.md`](../CLAUDE.md).

## 🎯 Empieza aquí

| Qué necesitas | Dónde |
|---|---|
| Crear o extender una feature de punta a punta | skill [`create-feature`](../.claude/skills/create-feature/SKILL.md) |
| Arquitectura y patrones del backend | [skills/api-backend-complete-guide.md](skills/api-backend-complete-guide.md) |
| Arquitectura y patrones del frontend | [skills/web-frontend-complete-guide.md](skills/web-frontend-complete-guide.md) |
| Formularios | [skills/form-creation-standard.md](skills/form-creation-standard.md) |
| Code-migrations | [backend/api-code-migration.md](backend/api-code-migration.md) |
| Permisos, roles y planes | [backend/permissions-system.md](backend/permissions-system.md) |
| Panel de administración | [backoffice/README.md](backoffice/README.md) |
| Decisiones de arquitectura | [adr/](adr/) |
| Glosario del dominio | [`../CONTEXT.md`](../CONTEXT.md) |

## 🗂️ Índice

```
docs/
├── README.md                              # ← estás aquí
├── skills/                                # guías de arquitectura + paso a paso
│   ├── README.md
│   ├── api-backend-complete-guide.md
│   ├── web-frontend-complete-guide.md
│   └── form-creation-standard.md
├── backend/
│   ├── README.md
│   ├── api-code-migration.md
│   └── permissions-system.md
├── frontend/
│   └── README.md
├── backoffice/
│   ├── README.md
│   ├── backoffice-architecture.md
│   ├── backoffice-domains-guide.md
│   └── backoffice-new-admin-feature.md
├── adr/
│   ├── 0001-clave-de-acceso-inmutable.md
│   └── 0002-aislamiento-por-tenant-en-la-capa-de-datos.md
└── superpowers/                           # planes y specs de features pasadas (histórico)
```

## 🎯 App Structure Overview

```
apps/api/             → Go backend (Gin + GORM), REST en /api/v1
apps/web/             → React, app de las clínicas (/clinical, /billing, /tasks...)
apps/backoffice/      → React, panel de administración de plataforma
apps/landing/         → Astro, landing page
apps/sri-xml-signer/  → Node.js, firma de XML para el SRI
packages/ui/          → @pengi/ui, componentes visuales compartidos
packages/shared/      → @pengi/shared, cliente HTTP e i18n compartidos
```

## 🤝 Contributing

Cuando descubras un patrón o una trampa, actualiza el `.md` que corresponde
(una sola fuente por tema). Si una doc contradice al código, gana el código:
corrige la doc.
