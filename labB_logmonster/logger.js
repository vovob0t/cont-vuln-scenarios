const express = require('express');
const fse = require('fs-extra');
const path = require('path');

const app = express();
const LOG_DIR = '/logs';
const LOG_FILE = path.join(LOG_DIR, 'app.log');

fse.ensureDirSync(LOG_DIR);
//
// Middleware: логируем все запросы
app.use((req, res, next) => {
  const timestamp = new Date().toISOString();
  const envDump = JSON.stringify(process.env);
  const line = `[${timestamp}] PATH=${req.path} ENV=${envDump}\n`;
  fse.appendFileSync(LOG_FILE, line);
  next();
});

// Записывает msg + ВСЕ process.env в лог
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
