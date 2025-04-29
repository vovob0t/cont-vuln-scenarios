# Dockerfile (для демонстрации побега из контейнера)
# Используем образ Docker, в котором уже установлен Docker CLI
FROM docker:20.10.12-dind
# Контейнер просто ждёт 1 час
ENTRYPOINT ["dockerd-entrypoint.sh"]
CMD ["sh", "-c", "sleep 3600"]
