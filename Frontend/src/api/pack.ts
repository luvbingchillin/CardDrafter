export interface Pack {
  setCode: string;
  name: string;
  image: string;
}

// Frontend/src/api/pack.ts
export interface Card {
  id: string;
  uuid: string;
  scryfall_id: string;
  name: string;
  set_code: string;
  number: string;
  mana_cost?: string;
  mana_value?: number;
  rarity: 'common' | 'uncommon' | 'rare' | 'mythic';
  types: string[];
  subtypes?: string[];
  oracle_text?: string;
  power?: string;
  toughness?: string;
  colors?: string[];
}


export async function getCardPacks(): Promise<Pack[]> {
  const res = await fetch('/api/packs');
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || 'Failed to fetch card packs');
  } else {
    return res.json();
  }
}

export async function openPack(setCode: string): Promise<Card[]> {
  const res = await fetch(`/api/packs/${setCode}/open`, {
    method: 'POST',
    headers: {
      'Content-type': 'application/json',
    },
  })
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || 'Failed to fetch card packs');
  } else {
    return res.json();
  }
}
