# Атакующий сервис: Node.js образ со скриптом для запроса к target
FROM node:14-slim
WORKDIR /app
COPY attacker.js /app/attacker.js
CMD sh -c "sleep 2 && exec node attacker.js"
