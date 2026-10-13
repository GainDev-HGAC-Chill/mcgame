FROM node:24-alpine AS registration-build
WORKDIR /registration
RUN corepack enable && corepack prepare pnpm@10.11.0 --activate
COPY frontend/registration/package.json frontend/registration/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/registration/ ./
RUN pnpm build

FROM nginx:1.27-alpine
COPY --from=registration-build /registration/dist/ /usr/share/nginx/html/
