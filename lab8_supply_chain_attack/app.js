 console.log("Приложение запущено...");
// Добавим простой HTTP-сервер на 8080
const http = require('http');
http.createServer((req, res) => res.end('OK')).listen(3000);
console.log('Listening on port 8080');

