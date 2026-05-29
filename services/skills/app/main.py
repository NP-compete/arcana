from fastapi import FastAPI

app = FastAPI(
    title="Arcana Skills",
    version="0.1.0",
    description="Skill registry, evolution loop, and execution engine",
)


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz")
async def readyz() -> dict[str, str]:
    return {"status": "ok"}
