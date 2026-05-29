FROM node:22-alpine AS builder
WORKDIR /app
COPY services/studio/package.json services/studio/package-lock.json* ./
RUN npm ci
COPY services/studio/ .
RUN npm run build

FROM nginx:1-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY services/studio/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 3000
