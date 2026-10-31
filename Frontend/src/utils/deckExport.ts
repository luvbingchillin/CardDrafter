import { type Card } from "../api/pack";

function convertToExport(cardList: Card[]) {
    let cardMap: Record<string, number> = {}
    for (const card of cardList) {
        const key = `${card.set_code.toUpperCase()} ${card.number} ${card.name}`
        if (cardMap[key]) {
            cardMap[key] += 1
        } else {
            cardMap[key] = 1
        }
    }
    const lines = ["Deck"]
    for (const [key, count] of Object.entries(cardMap)) {
        lines.push(`${count} ${key} `)
    }
    return lines.join("\n")
}

export async function copyToClipboard(cardList: Card[]): Promise<void> {
    const exportString = convertToExport(cardList)
    await navigator.clipboard.writeText(exportString);
}