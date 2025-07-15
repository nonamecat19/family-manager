FROM node:20-slim

WORKDIR /app

COPY package.json ./
COPY apps/service-finance/package.json ./apps/service-finance/
COPY apps/service-finance/tsconfig.json ./apps/service-finance/

COPY packages/db/package.json ./packages/db/
COPY packages/rabbitmq/package.json ./packages/rabbitmq/

RUN npm install

COPY apps/service-finance ./apps/service-finance
COPY packages/db ./packages/db
COPY packages/rabbitmq ./packages/rabbitmq

WORKDIR /app/apps/service-finance

RUN npm run build

EXPOSE 9000

CMD [ "npm", "run", "start:prod" ]
