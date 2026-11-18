import { type Card } from "../api/pack";

export function convertToExport(cardList: Card[]): string {
    const cardMap: Record<string, number> = {};

    for (const card of cardList) {
        // MTG Arena format: <Card Name> (<SET_CODE>) <CollectorNumber>
        const setCode = (card.set_code || 'DFT').toUpperCase();
        const colNum = card.number || '1';
        const key = `${card.name} (${setCode}) ${colNum}`;
        cardMap[key] = (cardMap[key] || 0) + 1;
    }

    const lines = ["Deck"];
    for (const [key, count] of Object.entries(cardMap)) {
        lines.push(`${count} ${key}`);
    }

    return lines.join("\n");
}

export async function copyToClipboard(cardList: Card[]): Promise<void> {
    const exportString = convertToExport(cardList);
    await navigator.clipboard.writeText(exportString);
}
