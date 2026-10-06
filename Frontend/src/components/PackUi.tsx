
import styles from './PackUi.module.css';
import type { Pack } from '../api/pack';


interface PackUiProps {
    pack: Pack;
    onClick?: () => void; // Allow parent to pass a click handler!
}


export function PackUi({ pack, onClick }: PackUiProps) {
    return (
        <div key={pack.setCode} className={styles.packCard} onClick={onClick}>
            <div className={styles.imageWrapper}>
                <img
                    src={pack.image}
                    alt={pack.name}
                    className={styles.packImage}
                />
            </div>
            <div className={styles.packInfo}>
                <p className={styles.packName}>{pack.name}</p>
                <span className={styles.setCode}>{pack.setCode}</span>

            </div>
        </div>
    )
}