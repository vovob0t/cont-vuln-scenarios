// Простое Express-приложение, которое выводит сообщение
var express = require('express');
var app = express();

app.get('/', function(req, res) {
  res.send('Hello from vulnerable Express 3.0.0 app!');
});

app.listen(3000, function() {
  console.log('App listening on port 3000');
});
