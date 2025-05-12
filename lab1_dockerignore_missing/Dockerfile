# Dockerfile (уязвимый вариант)
FROM python:3.9-slim
WORKDIR /app
# Копируются все файлы из контекста сборки, включая .env, references и прочее
COPY . .
RUN pip install flask
CMD ["python", "app/app.py"]
