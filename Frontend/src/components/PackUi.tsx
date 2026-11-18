import styles from './PackUi.module.css';
import type { Pack } from '../api/pack';

interface PackUiProps {
    pack: Pack;
    onClick?: () => void;
}

export function PackUi({ pack, onClick }: PackUiProps) {
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            onClick?.();
        }
    };

    return (
        <div
            role="button"
            tabIndex={0}
            aria-label={`Open ${pack.name} booster pack`}
            className={styles.packCard}
            onClick={onClick}
            onKeyDown={handleKeyDown}
        >
            <div className={styles.imageWrapper}>
                <img
                    src={pack.image}
                    alt={`${pack.name} booster artwork`}
                    className={styles.packImage}
                    loading="lazy"
                />
            </div>
            <div className={styles.packInfo}>
                <p className={styles.packName}>{pack.name}</p>
                <span className={styles.setCode}>{pack.setCode}</span>
            </div>
        </div>
    );
}
