# --- builder ---
FROM node:20-bookworm-slim AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install --no-audit --no-fund
COPY . .
ARG VITE_API_BASE_URL
ARG VITE_NEW_RELIC_LICENSE_KEY
ARG VITE_NEW_RELIC_APP_ID
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_NEW_RELIC_LICENSE_KEY=$VITE_NEW_RELIC_LICENSE_KEY
ENV VITE_NEW_RELIC_APP_ID=$VITE_NEW_RELIC_APP_ID
RUN npm run build

# --- runtime: nginx ---
FROM nginx:1.27-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
