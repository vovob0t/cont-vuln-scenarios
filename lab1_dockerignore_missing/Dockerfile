# Dockerfile (уязвимый вариант)
FROM python:3.9-slim
WORKDIR /app
# Копируются все файлы из контекста сборки, включая .env, references и прочее
COPY . /app
RUN pip install flask
CMD ["python", "app.py"]
