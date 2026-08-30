# Web 构建与 Nginx 静态托管。/api 与 /files 由 nginx.conf 反代给 API 容器。
FROM node:24.20.0-alpine AS build
WORKDIR /src
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm ci
COPY web/admin/ .
RUN npm run build

FROM nginx:1.27-alpine
COPY deploy/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
