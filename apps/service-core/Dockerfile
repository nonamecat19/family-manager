FROM node:20-slim

WORKDIR /app

COPY package.json ./
COPY apps/service-core/package.json ./apps/service-core/
COPY apps/service-core/tsconfig.json ./apps/service-core/

COPY packages/db/package.json ./packages/db/
COPY packages/rabbitmq/package.json ./packages/rabbitmq/

RUN npm install

COPY apps/service-core ./apps/service-core
COPY packages/db ./packages/db
COPY packages/rabbitmq ./packages/rabbitmq

WORKDIR /app/apps/service-core

RUN npm run build

EXPOSE 9000

CMD [ "npm", "run", "start:prod" ]
