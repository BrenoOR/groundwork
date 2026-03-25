from fastapi import FastAPI

from app.api.routes import files, users

app = FastAPI(
    title="Team Files API",
    description="Users can browse their personal S3 folder and their teams' shared folders.",
    version="0.1.0",
)

app.include_router(users.router)
app.include_router(files.router)


@app.get("/health", tags=["health"])
def health() -> dict[str, str]:
    return {"status": "ok"}
