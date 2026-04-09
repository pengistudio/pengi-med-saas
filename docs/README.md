# Pengi Med SaaS — Development Documentation

Central documentation hub for all developers working on Pengi Med SaaS.

**Whether you use Claude Code, Cursor, Copilot, VS Code, or anything else — this is where to find development guides.**

## 📚 Documentation by Area

### Backend (Go)
**Path:** [`apps/api/`](../apps/api)

See [backend/README.md](backend/README.md) for:
- Architecture & design patterns
- How to implement new features
- Database migrations
- Error handling & responses
- Permissions & authorization

Start here: [backend/api-architecture-reference.md](backend/api-architecture-reference.md)

### Frontend (React — SaaS App)
**Path:** [`apps/web/`](../apps/web)

See [frontend/README.md](frontend/README.md) for:
- Frontend architecture & patterns
- API services integration
- State management (Zustand)
- How to implement new features
- Routing, permissions & features

Start here: [frontend/web-architecture-reference.md](frontend/web-architecture-reference.md)

### Backoffice (React — Admin Panel)
**Path:** [`apps/backoffice/`](../apps/backoffice)

See [backoffice/README.md](backoffice/README.md) for:
- Admin panel architecture
- Managing companies, plans, subscriptions
- Roles, permissions, features
- How to add new admin operations
- Automatic features calculation

Start here: [backoffice/backoffice-architecture.md](backoffice/backoffice-architecture.md)

### Database
**Path:** [`apps/api/migrations/`](../apps/api/migrations)

See [database/README.md](database/README.md) for:
- Migration strategy
- Schema documentation
- Multi-tenancy design

*Documentation coming soon*

## 🔗 IDE Integrations

- **Claude Code** — Reads `/docs` automatically
- **Cursor, VS Code, Others** — Use `/docs` folder directly (no setup needed)

All documentation is IDE-agnostic and lives here.

## 📖 Quick Navigation

### Backend Tasks
| What do you need? | Where to look |
|---|---|
| Understand the backend architecture | [backend/api-architecture-reference.md](backend/api-architecture-reference.md) |
| Add a new API feature | [backend/api-new-feature.md](backend/api-new-feature.md) |
| Create a database migration | [backend/api-code-migration.md](backend/api-code-migration.md) |
| Handle errors & responses | [backend/api-errors-routes-responses.md](backend/api-errors-routes-responses.md) |
| Manage permissions system | [backend/permissions-system.md](backend/permissions-system.md) |

### Frontend Tasks
| What do you need? | Where to look |
|---|---|
| Understand frontend architecture | [frontend/web-architecture-reference.md](frontend/web-architecture-reference.md) |
| Add a new frontend feature | [frontend/web-new-feature.md](frontend/web-new-feature.md) |
| Create API services | [frontend/web-api-services.md](frontend/web-api-services.md) |
| Use state management | [frontend/web-state-management.md](frontend/web-state-management.md) |

### Backoffice Tasks
| What do you need? | Where to look |
|---|---|
| Understand backoffice architecture | [backoffice/backoffice-architecture.md](backoffice/backoffice-architecture.md) |
| Manage companies, plans, subscriptions | [backoffice/backoffice-domains-guide.md](backoffice/backoffice-domains-guide.md) |
| Add a new admin operation | [backoffice/backoffice-new-admin-feature.md](backoffice/backoffice-new-admin-feature.md) |
| Understand features calculation | [backoffice/backoffice-domains-guide.md](backoffice/backoffice-domains-guide.md#-diagrama-de-relaciones) |

### General
| What do you need? | Where to look |
|---|---|
| General project rules | [../CLAUDE.md](../CLAUDE.md) |

## 🤝 Contributing

When you discover patterns, best practices, or gotchas:
1. Update the relevant `.md` file in `/docs`
2. Keep documentation in sync — single source of truth
3. Add examples, not just theory

## 📞 Questions?

- Check `CLAUDE.md` for project conventions
- Search this folder for related topics
- Update docs if something is missing or outdated

## 🗂️ Full Index

```
docs/
├── README.md                          # ← You are here
│
├── backend/
│   ├── README.md
│   ├── api-architecture-reference.md
│   ├── api-new-feature.md
│   ├── api-code-migration.md
│   ├── api-errors-routes-responses.md
│   └── permissions-system.md
│
├── frontend/
│   ├── README.md
│   ├── web-architecture-reference.md
│   ├── web-api-services.md
│   ├── web-state-management.md
│   └── web-new-feature.md
│
├── backoffice/
│   ├── README.md
│   ├── backoffice-architecture.md
│   ├── backoffice-domains-guide.md
│   └── backoffice-new-admin-feature.md
│
└── database/
    └── (coming soon)
```

## 🎯 App Structure Overview

```
Pengi Med SaaS
├── apps/api/                 → Go backend (Gin + GORM)
│   └── endpoints /api/v1/... → REST API
│
├── apps/web/                 → React frontend (SaaS app for users)
│   └── routes                → /dashboard, /clinical, /billing, etc
│
├── apps/backoffice/          → React admin panel
│   └── routes                → /companies, /plans, /subscriptions, etc
│
├── apps/landing/             → Astro landing page
│
└── docs/                      → This documentation
```

## 🚀 Getting Started

1. **New to the project?** Start with `CLAUDE.md` for project conventions
2. **Working on backend?** Read [backend/api-architecture-reference.md](backend/api-architecture-reference.md)
3. **Working on frontend?** Read [frontend/web-architecture-reference.md](frontend/web-architecture-reference.md)
4. **Working on admin panel?** Read [backoffice/backoffice-architecture.md](backoffice/backoffice-architecture.md)
