import logging
from concurrent import futures

import grpc

try:
    from grpc_health_v1 import health, health_pb2, health_pb2_grpc
    _HEALTH_AVAILABLE = True
except ImportError:
    _HEALTH_AVAILABLE = False
    health = None
    health_pb2 = None
    health_pb2_grpc = None

logger = logging.getLogger(__name__)


class HealthServicer:
    def Check(self, request, context):
        if not _HEALTH_AVAILABLE:
            context.set_code(grpc.StatusCode.UNIMPLEMENTED)
            context.set_details("Health check not available")
            return None
        return health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.SERVING
        )

    def Watch(self, request, context):
        if not _HEALTH_AVAILABLE:
            context.set_code(grpc.StatusCode.UNIMPLEMENTED)
            return
        yield health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.SERVING
        )


def create_grpc_server(port: int = 50051, max_workers: int = 10) -> grpc.Server:
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=max_workers),
        maximum_concurrent_rpcs=100,
    )

    if _HEALTH_AVAILABLE and health_pb2_grpc is not None:
        health_servicer = HealthServicer()
        health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
        logger.info("gRPC health service registered")
    else:
        logger.warning("gRPC health service unavailable (install grpcio-health-checking)")

    server.add_insecure_port(f"[::]:{port}")
    logger.info(f"gRPC server starting on port {port}")

    return server


def serve_grpc(port: int = 50051):
    server = create_grpc_server(port)
    server.start()
    logger.info(f"gRPC server listening on 0.0.0.0:{port}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve_grpc()
