FROM node:20-slim

WORKDIR /app

COPY package.json ./
COPY apps/service-iot/package.json ./apps/service-iot/
COPY apps/service-iot/tsconfig.json ./apps/service-iot/

COPY packages/db/package.json ./packages/db/
COPY packages/rabbitmq/package.json ./packages/rabbitmq/

RUN npm install

COPY apps/service-iot ./apps/service-iot
COPY packages/db ./packages/db
COPY packages/rabbitmq ./packages/rabbitmq

WORKDIR /app/apps/service-iot

RUN npm run build

EXPOSE 9005

CMD [ "npm", "run", "start:prod" ]
