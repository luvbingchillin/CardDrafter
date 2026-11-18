import type { Card } from "../api/pack";
import styles from './CardUi.module.css';

interface CardUiProps {
    card: Card;
    onClick?: () => void;
    isSelected?: boolean;
}

export function CardUi({ card, onClick, isSelected }: CardUiProps) {
    const imageUrl = `https://api.scryfall.com/cards/${card.scryfall_id}?format=image&version=normal`;

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (onClick && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault();
            onClick();
        }
    };

    const isInteractive = Boolean(onClick);

    return (
        <div
            role={isInteractive ? "button" : undefined}
            tabIndex={isInteractive ? 0 : undefined}
            aria-label={isInteractive ? `Card: ${card.name}, ${card.rarity}` : undefined}
            aria-pressed={isInteractive ? isSelected : undefined}
            onClick={onClick}
            onKeyDown={handleKeyDown}
            className={`${styles.cardWrapper} ${styles[card.rarity]} ${isInteractive ? styles.interactive : ''} ${isSelected ? styles.selected : ''}`}
        >
            <img
                src={imageUrl}
                alt={card.name}
                className={styles.cardImage}
                loading="lazy"
                onError={(e) => {
                    // Fallback to default card back placeholder if Scryfall image request fails
                    (e.currentTarget as HTMLImageElement).src = '/packs/default.png';
                }}
            />
        </div>
    );
}
