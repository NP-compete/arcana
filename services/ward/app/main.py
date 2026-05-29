from fastapi import FastAPI

app = FastAPI(
    title="Arcana Ward",
    version="0.1.0",
    description="Guardrails, policy enforcement, and safety pipeline",
)


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz")
async def readyz() -> dict[str, str]:
    return {"status": "ok"}
