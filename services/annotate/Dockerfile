FROM python:3.12-slim AS builder
WORKDIR /app
COPY services/annotate/pyproject.toml .
COPY services/annotate/app/ app/
RUN pip install --no-cache-dir .

FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY services/annotate/app/ app/
USER 65534:65534
EXPOSE 8102
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8102"]
