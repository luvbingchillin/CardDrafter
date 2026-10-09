import logging
from concurrent import futures
from contextlib import asynccontextmanager
import grpc
from fastapi import FastAPI

from app.routers.packs import router as packs_router
from app.gen import analytics_pb2_grpc
from app.services.grpc_service import AnalyticsServicer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # --- Startup: Start gRPC Server on port 50051 ---
    grpc_server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    analytics_pb2_grpc.add_AnalyticsServiceServicer_to_server(AnalyticsServicer(), grpc_server)
    
    # Listen on all container interfaces (0.0.0.0) on port 50051
    grpc_server.add_insecure_port("[::]:50051")
    grpc_server.start()
    logger.info("gRPC server listening on port 50051...")

    yield  # FastAPI runs and serves HTTP traffic here

    # --- Shutdown: Gracefully stop gRPC server ---
    logger.info("Stopping gRPC server...")
    grpc_server.stop(grace=5)


app = FastAPI(
    title="Card Simulator Analytics Service",
    description="Microservice for statistical modeling, EV calculation, and draft analytics",
    version="1.0.0",
    lifespan=lifespan,
)


@app.get("/health")
async def health_check():
    return {"status": "ok", "service": "analytics"}


app.include_router(packs_router, prefix="/api/v1/packs", tags=["Packs"])
