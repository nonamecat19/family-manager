FROM node:20-slim

WORKDIR /app

COPY package.json ./
COPY apps/service-events/package.json ./apps/service-events/
COPY apps/service-events/tsconfig.json ./apps/service-events/

COPY packages/db/package.json ./packages/db/
COPY packages/rabbitmq/package.json ./packages/rabbitmq/

RUN npm install

COPY apps/service-events ./apps/service-events
COPY packages/db ./packages/db
COPY packages/rabbitmq ./packages/rabbitmq

WORKDIR /app/apps/service-events

RUN npm run build

EXPOSE 9000

CMD [ "npm", "run", "start:prod" ]
