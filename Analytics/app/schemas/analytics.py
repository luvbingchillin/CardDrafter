from typing import List, Optional
from pydantic import BaseModel, Field


class PackCardInput(BaseModel):
    """Represents a card's data passed for analytics calculation."""
    scryfall_id: str
    name: str
    rarity: str
    price_usd: Optional[float] = Field(default=None, description="Current market price in USD")


class EVCalculationRequest(BaseModel):
    """Input payload from Go backend requesting EV calculation."""
    set_code: str
    cards: List[PackCardInput]


class EVCalculationResponse(BaseModel):
    """Output metrics returned back to Go backend."""
    set_code: str
    total_cards_analyzed: int
    average_pack_ev: float
    mythic_ev: float
    rare_ev: float
    uncommon_ev: float
    common_ev: float
