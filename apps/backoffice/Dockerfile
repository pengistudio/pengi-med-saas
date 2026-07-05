# Build stage
FROM node:24-alpine AS builder
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json
COPY apps/backoffice/package.json apps/backoffice/package.json
COPY packages/ui/package.json packages/ui/package.json
RUN --mount=type=cache,id=pnpm-backoffice,target=/pnpm/store pnpm install --frozen-lockfile --filter backoffice...
COPY packages/ui packages/ui
COPY apps/backoffice apps/backoffice
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
ARG VITE_WEB_APP_URL
ENV VITE_WEB_APP_URL=$VITE_WEB_APP_URL
RUN pnpm --filter backoffice run build

# Serve stage
FROM nginx:alpine
COPY --from=builder /app/apps/backoffice/dist /usr/share/nginx/html
COPY apps/backoffice/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
