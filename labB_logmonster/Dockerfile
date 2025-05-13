FROM node:16-slim
RUN apt-get update && apt-get install -y docker.io \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json .
RUN npm install

COPY logger.js .

EXPOSE 4000
CMD ["node", "logger.js"]
