import type { Card } from "../api/pack";
import styles from './CardUi.module.css'

interface CardUiProps {
    card: Card;
    onClick?: () => void;
}


export function CardUi(prop: CardUiProps) {
    const card = prop.card
    const imageUrl = `https://api.scryfall.com/cards/${card.scryfall_id}?format=image&version=normal`;
    return (
        <div className={`${styles.cardWrapper} ${styles[card.rarity]}`}>
            <img
                src={imageUrl}
                alt={card.name}
                className={styles.cardImage}
                loading="lazy"
            />
        </div>

    )
}