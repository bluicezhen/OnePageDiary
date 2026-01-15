FROM node:24-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY docker-entrypoint.d/90-config.sh /docker-entrypoint.d/90-config.sh
RUN chmod +x /docker-entrypoint.d/90-config.sh
