FROM node:22-alpine AS builder
WORKDIR /app
COPY services/studio/package.json services/studio/package-lock.json* ./
RUN npm ci
COPY services/studio/ .
RUN npm run build

FROM nginx:1-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY services/studio/nginx.conf /etc/nginx/conf.d/default.conf
RUN chown -R 101:101 /usr/share/nginx/html && \
    chown -R 101:101 /var/cache/nginx && \
    chown -R 101:101 /var/log/nginx && \
    touch /var/run/nginx.pid && chown 101:101 /var/run/nginx.pid
USER 101
EXPOSE 3000
