# Dockerfile (уязвимый вариант)
FROM python:3.9-slim
WORKDIR /app
COPY . /app
RUN pip install flask
# Здесь не указан пользователь – по умолчанию процесс работает от root
CMD ["python", "app.py"]
