# Используется подозрительный базовый образ, которого может не существовать в официальном реестре
FROM vulnerable-base:latest
WORKDIR /app
COPY app.js /app/app.js
EXPOSE 3000
CMD ["node", "app.js"]
