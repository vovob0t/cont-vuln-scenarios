# Целевой сервис: простой HTTP-сервер на Python, отдающий содержимое data.txt
FROM python:3.9-slim
WORKDIR /app
COPY data.txt /app/data.txt
RUN pip install flask
COPY server.py /app/server.py
EXPOSE 4000
CMD ["python", "server.py"]
