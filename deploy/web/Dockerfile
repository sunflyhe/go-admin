# Web 构建与 Nginx 静态托管:管理端 /admin/、应用端根路径;
# /api、/admin-api 与 /files 由 nginx.conf 反代给 API 容器。
FROM node:24.20.0-alpine AS build-admin
WORKDIR /src
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm ci
COPY web/admin/ .
RUN npm run build

FROM node:24.20.0-alpine AS build-app
WORKDIR /src
COPY web/app/package.json web/app/package-lock.json ./
RUN npm ci
COPY web/app/ .
RUN npm run build

FROM nginx:1.27-alpine
COPY deploy/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build-admin /src/dist /usr/share/nginx/admin
COPY --from=build-app /src/dist /usr/share/nginx/html
EXPOSE 80
