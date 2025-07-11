FROM node:20-slim

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable

WORKDIR /app

COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY apps/service-core/package.json ./apps/service-core/
COPY packages/db/package.json ./packages/db/
COPY packages/rabbitmq/package.json ./packages/rabbitmq/

RUN pnpm install

COPY apps/service-core ./apps/service-core
COPY packages/db ./packages/db
COPY packages/rabbitmq ./packages/rabbitmq

WORKDIR /app/apps/service-core

RUN pnpm run build

CMD [ "npm", "run", "start:dev" ]
