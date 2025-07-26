FROM node:20-slim

WORKDIR /app

COPY package.json ./
COPY apps/service-iot/package.json ./apps/client-iot/
COPY apps/client-iot/tsconfig.json ./apps/client-iot/

COPY packages/db/package.json ./packages/db/
COPY packages/rabbitmq/package.json ./packages/rabbitmq/

RUN npm install

COPY apps/client-iot ./apps/client-iot
COPY packages/db ./packages/db
COPY packages/rabbitmq ./packages/rabbitmq

WORKDIR /app/apps/client-iot

RUN npm run build

EXPOSE 8005

CMD [ "npm", "run", "start:prod" ]
