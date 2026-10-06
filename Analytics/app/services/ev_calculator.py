from typing import List
import pandas as pd
from app.schemas.analytics import PackCardInput, EVCalculationResponse


def calculate_pack_ev(set_code: str, cards: List[PackCardInput]) -> EVCalculationResponse:
    """
    Calculates the statistical Expected Value (EV) of a booster pack.
    
    Formula:
      - 1 Rare slot = (1/8 * Mythic EV) + (7/8 * Rare EV)
      - 3 Uncommon slots = 3 * Uncommon EV
      - 10 Common slots = 10 * Common EV
    """
    if not cards:
        return EVCalculationResponse(
            set_code=set_code,
            total_cards_analyzed=0,
            average_pack_ev=0.0,
            mythic_ev=0.0,
            rare_ev=0.0,
            uncommon_ev=0.0,
            common_ev=0.0,
        )

    # Convert card list to Pandas DataFrame for high-performance vector math
    df = pd.DataFrame([card.model_dump() for card in cards])

    # Fill missing prices with 0.0
    df["price_usd"] = df["price_usd"].fillna(0.0)

    # Calculate average price per rarity
    rarity_means = df.groupby("rarity")["price_usd"].mean().to_dict()

    mythic_mean = float(rarity_means.get("mythic", 0.0))
    rare_mean = float(rarity_means.get("rare", 0.0))
    uncommon_mean = float(rarity_means.get("uncommon", 0.0))
    common_mean = float(rarity_means.get("common", 0.0))

    # Standard MTG Collation Formula:
    # Rare slot: 1-in-8 chance of Mythic (12.5%), 7-in-8 chance of Rare (87.5%)
    rare_slot_ev = (0.125 * mythic_mean) + (0.875 * rare_mean)
    uncommon_slots_ev = 3.0 * uncommon_mean
    common_slots_ev = 10.0 * common_mean

    total_pack_ev = rare_slot_ev + uncommon_slots_ev + common_slots_ev

    return EVCalculationResponse(
        set_code=set_code,
        total_cards_analyzed=len(df),
        average_pack_ev=round(total_pack_ev, 2),
        mythic_ev=round(mythic_mean, 2),
        rare_ev=round(rare_mean, 2),
        uncommon_ev=round(uncommon_mean, 2),
        common_ev=round(common_mean, 2),
    )
