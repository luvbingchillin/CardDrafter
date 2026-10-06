from fastapi import APIRouter
from app.schemas.analytics import EVCalculationRequest, EVCalculationResponse
from app.services.ev_calculator import calculate_pack_ev

router = APIRouter()


@router.post("/calculate-ev", response_model=EVCalculationResponse)
async def compute_ev(payload: EVCalculationRequest):
    """
    Computes statistical Expected Value (EV) for a given set's card pool.
    Called internally by the Go backend.
    """
    return calculate_pack_ev(payload.set_code, payload.cards)
