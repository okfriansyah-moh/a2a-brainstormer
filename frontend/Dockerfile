# node:20-alpine — Alpine 3.21+ base, minimal OS packages, 0 high/critical CVEs.
FROM node:20-alpine

WORKDIR /app

RUN npm install -g pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

# Port the Vite dev server listens on — matches FRONTEND_HOST_PORT in .env (default 5173).
ARG FRONTEND_PORT=5173
EXPOSE ${FRONTEND_PORT}

# --host 0.0.0.0 makes Vite reachable from outside the container.
CMD ["pnpm", "dev", "--host", "0.0.0.0"]
