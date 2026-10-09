import grpc
from concurrent import futures
import logging

from app.gen import analytics_pb2
from app.gen import analytics_pb2_grpc
from app.schemas.analytics import PackCardInput
from app.services.ev_calculator import calculate_pack_ev

logger = logging.getLogger(__name__)


class AnalyticsServicer(analytics_pb2_grpc.AnalyticsServiceServicer):
    """
    Implements the gRPC contract defined in proto/analytics.proto.
    """

    def CalculatePackEV(self, request: analytics_pb2.EVRequest, context: grpc.ServicerContext) -> analytics_pb2.EVResponse:
        """
        Receives an EVRequest containing set_code and a repeated list of CardData,
        runs the statistical calculations, and returns an EVResponse.
        """
        try:
            # 1. Map incoming Protobuf CardData messages to our domain/Pydantic schema
            domain_cards = [
                PackCardInput(
                    scryfall_id=c.scryfall_id,
                    name=c.name,
                    rarity=c.rarity,
                    price_usd=c.price_usd if c.price_usd > 0 else None,
                )
                for c in request.cards
            ]

            # 2. Invoke our existing calculation engine
            result = calculate_pack_ev(request.set_code, domain_cards)

            # 3. Pack the calculated metrics into the generated Protobuf Response object
            return analytics_pb2.EVResponse(
                set_code=result.set_code,
                total_cards_analyzed=result.total_cards_analyzed,
                average_pack_ev=result.average_pack_ev,
                mythic_ev=result.mythic_ev,
                rare_ev=result.rare_ev,
                uncommon_ev=result.uncommon_ev,
                common_ev=result.common_ev,
            )

        except Exception as e:
            logger.error(f"Error calculating pack EV via gRPC: {e}")
            # In gRPC, errors are signaled via status codes on the context object
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return analytics_pb2.EVResponse()
