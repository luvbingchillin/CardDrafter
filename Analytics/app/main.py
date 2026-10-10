import logging
from concurrent import futures
from contextlib import asynccontextmanager
import grpc
from fastapi import FastAPI

from app.routers.packs import router as packs_router
from app.gen import analytics_pb2_grpc
from app.services.grpc_service import AnalyticsServicer
import threading
from app.services.stream_consumer import run_stream_consumer

consumer_stop_event = threading.Event()

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Manages application startup and graceful shutdown hooks:
    - Starts internal gRPC server on port 50051 for on-demand queries.
    - Spawns background Redis Stream consumer worker.
    """
    # 1. Startup: Launch gRPC Server
    grpc_server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    analytics_pb2_grpc.add_AnalyticsServiceServicer_to_server(AnalyticsServicer(), grpc_server)
    grpc_server.add_insecure_port("[::]:50051")
    grpc_server.start()
    logger.info("gRPC server started on port 50051")

    # 2. Startup: Launch Redis Stream Worker Thread
    consumer_thread = threading.Thread(
        target=run_stream_consumer,
        args=(consumer_stop_event,),
        daemon=True,
    )
    consumer_thread.start()
    logger.info("Redis stream consumer worker started")

    yield

    # 3. Shutdown: Terminate background worker and gRPC server
    logger.info("Initiating graceful shutdown...")
    consumer_stop_event.set()
    consumer_thread.join(timeout=3)
    grpc_server.stop(grace=5)
    logger.info("Analytics service stopped successfully")


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
