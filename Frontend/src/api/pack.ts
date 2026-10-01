export interface Pack {
  setCode: string;
  name: string;
  image: string;
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
