FROM node:24-bookworm-slim AS build
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY common/ ../common/
COPY web/ ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build -- --webpack

FROM node:24-bookworm-slim AS runtime
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 HOSTNAME=0.0.0.0 PORT=3000
COPY --from=build --chown=node:node /app/web/.next/standalone/ ./
COPY --from=build --chown=node:node /app/web/.next/static/ ./web/.next/static/
USER node
CMD ["node", "web/server.js"]
