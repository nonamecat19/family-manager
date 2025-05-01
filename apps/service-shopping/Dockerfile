FROM node:20-alpine AS build

WORKDIR /app

COPY package*.json ./
RUN npm i -f

COPY . .
RUN npm run build

FROM node:20-alpine

WORKDIR /app

COPY --from=build /app/package*.json ./
COPY --from=build /app/dist ./dist
COPY --from=build /app/node_modules ./node_modules

EXPOSE 3001

CMD ["npm", "run", "start:prod"]