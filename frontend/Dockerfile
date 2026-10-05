# Immagine del frontend: server Next.js in modalità standalone.

# --- Dipendenze ---
FROM node:22.23.1-alpine3.24@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2 AS deps

WORKDIR /app

# Solo le dipendenze di produzione, dalle versioni esatte di package-lock.json.
COPY package.json package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund

# --- Build ---
FROM node:22.23.1-alpine3.24@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2 AS build

WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Lo script prebuild copia font e icone di Bootstrap Italia in public/.
RUN npm run build

# --- Esecuzione ---
FROM node:22.23.1-alpine3.24@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2

WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

# Il server standalone non include public/ e .next/static: si copiano accanto.
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public

# Utente senza privilegi già definito dall'immagine Node.
USER node

EXPOSE 3000

CMD ["node", "server.js"]
