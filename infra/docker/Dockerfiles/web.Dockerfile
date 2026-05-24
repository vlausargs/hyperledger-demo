FROM node:22-alpine AS builder
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app
COPY packages/web/package.json packages/web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY packages/web/ ./
RUN pnpm build

FROM node:22-alpine
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
ENV NODE_ENV=production
CMD ["node", "build"]
