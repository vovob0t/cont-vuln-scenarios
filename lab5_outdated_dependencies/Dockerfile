# Уязвимый Dockerfile – используется Node.js образ с устаревшими зависимостями
FROM node:14-slim
WORKDIR /app
COPY package.json .
RUN npm install
COPY . .
EXPOSE 3000
CMD ["node", "app.js"]
