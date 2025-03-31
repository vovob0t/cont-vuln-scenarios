// Простой скрипт на Node.js, который делает HTTP-запрос к целевому сервису и выводит ответ
const http = require('http');

http.get('http://target:4000/', (res) => {
  let data = '';
  res.on('data', chunk => data += chunk);
  res.on('end', () => console.log("Полученные данные из target:", data));
}).on('error', (err) => {
  console.error("Ошибка: ", err.message);
});
