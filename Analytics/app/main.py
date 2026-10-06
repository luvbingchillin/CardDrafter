from fastapi import FastAPI
from app.routers.packs import router as packs_router

app = FastAPI(
    title="Card Simulator Analytics Service",
    description="Microservice for statistical modeling, EV calculation, and draft analytics",
    version="1.0.0",
)


@app.get("/health")
async def health_check():
    """Health check endpoint for Docker container liveness probes."""
    return {"status": "ok", "service": "analytics"}


# Mount the packs analytics router under /api/v1/packs
app.include_router(packs_router, prefix="/api/v1/packs", tags=["Packs"])
