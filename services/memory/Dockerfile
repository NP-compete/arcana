FROM python:3.12-slim AS builder
WORKDIR /app
COPY services/memory/pyproject.toml .
COPY services/memory/app/ app/
RUN pip install --no-cache-dir .

FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY services/memory/app/ app/
USER 65534:65534
EXPOSE 8087
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8087", "--timeout-graceful-shutdown", "30"]
