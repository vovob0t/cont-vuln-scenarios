# Dockerfile (уязвимый вариант)
FROM python:3.9-slim
WORKDIR /app
COPY . .
RUN pip install flask
# Здесь не указан пользователь – по умолчанию процесс работает от root
CMD ["python", "app/app.py"]
