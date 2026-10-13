import { useParams, useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { type Card, type SetAnalytics, openPack, getSetAnalytics } from '../api/pack';
import { CardUi } from '../components/CardUi';
import styles from './PackOpeningPage.module.css';

function preloadImage(url: string): Promise<void> {
    return new Promise((resolve) => {
        const img = new Image();
        img.src = url;
        img.onload = () => resolve();
        img.onerror = () => resolve();
    });
}

export function PackOpeningPage() {
    const { setCode } = useParams<{ setCode: string }>();
    const navigate = useNavigate();
    const [cards, setCards] = useState<Card[]>([]);
    const [analytics, setAnalytics] = useState<SetAnalytics | null>(null);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (!setCode) return;
        const getSetAnalyticFunc = async () => {
            try {
                const response = await getSetAnalytics(setCode);
                setAnalytics(response);
            } catch (err) {
                console.error(err);
            }
        };
        getSetAnalyticFunc();
    }, [setCode]);

    const openPackBySet = async () => {
        if (!setCode || loading) return;
        try {
            setLoading(true);
            setError('');
            // Reset cards immediately so the opening/buffering screen takes over smoothly
            setCards([]);

            const openedCards = await openPack(setCode);

            // Preload all 14 card images in parallel so they display simultaneously
            const imagePromises = openedCards.map((card) => {
                const url = `https://api.scryfall.com/cards/${card.scryfall_id}?format=image&version=normal`;
                return preloadImage(url);
            });
            await Promise.all(imagePromises);

            setCards(openedCards);
            setAnalytics((prev) => {
                if (!prev) return prev;
                let mythics = 0, rares = 0, uncommons = 0, commons = 0;
                for (const c of openedCards) {
                    if (c.rarity === 'mythic') mythics++;
                    else if (c.rarity === 'rare') rares++;
                    else if (c.rarity === 'uncommon') uncommons++;
                    else if (c.rarity === 'common') commons++;
                }
                return {
                    ...prev,
                    total_packs_opened: prev.total_packs_opened + 1,
                    mythic_count: prev.mythic_count + mythics,
                    rare_count: prev.rare_count + rares,
                    uncommon_count: prev.uncommon_count + uncommons,
                    common_count: prev.common_count + commons,
                };
            });
        } catch (err) {
            if (err instanceof Error) {
                setError(err.message);
            } else {
                setError('Failed to open pack');
            }
        } finally {
            setLoading(false);
        }
    };

    const packImageUrl = `/packs/${setCode?.toUpperCase()}.png`;

    return (
        <div className={styles.container}>
            {/* Navigation & Header */}
            <div className={styles.topBar}>
                <button className={styles.backButton} onClick={() => navigate('/packs')}>
                    ← Back to Sets
                </button>
                <h1 className={styles.setTitle}>{setCode?.toUpperCase()} Booster Pack</h1>
                {analytics && (
                    <div className={styles.analyticsHud}>
                        <span>Opened: <strong>{analytics.total_packs_opened}</strong></span>
                        <span>Mythics: <strong className={styles.mythicText}>{analytics.mythic_count}</strong></span>
                        <span>Rares: <strong className={styles.rareText}>{analytics.rare_count}</strong></span>
                    </div>
                )}
            </div>

            {/* Persistent Pack Action Bar */}
            <div className={styles.actionBar}>
                <div className={styles.packIdentity}>
                    <img
                        src={packImageUrl}
                        alt={`${setCode} booster`}
                        className={styles.miniPackThumb}
                        onError={(e) => {
                            (e.currentTarget as HTMLImageElement).src = '/packs/default.png';
                        }}
                    />
                    <div className={styles.packDetails}>
                        <span className={styles.packName}>{setCode?.toUpperCase()} Booster</span>
                        {cards.length > 0 && (
                            <span className={styles.cardCountText}>{cards.length} Cards Revealed</span>
                        )}
                    </div>
                </div>

                <button
                    className={styles.openButton}
                    onClick={openPackBySet}
                    disabled={loading}
                >
                    {loading
                        ? 'Cracking & Buffering...'
                        : cards.length > 0
                        ? 'Rip Another Pack'
                        : 'Open Booster'}
                </button>
            </div>

            {error && <div className={styles.errorBanner}>{error}</div>}

            {/* Stage Area: Loading Buffer, Cards Grid, or Initial Prompt */}
            <div className={styles.arena}>
                {loading ? (
                    <div className={styles.loadingContainer}>
                        <div className={styles.spinner}></div>
                        <p className={styles.loadingText}>Cracking pack & buffering card artwork...</p>
                    </div>
                ) : cards.length > 0 ? (
                    <div className={styles.cardGrid}>
                        {cards.map((card) => (
                            <CardUi key={card.id} card={card} />
                        ))}
                    </div>
                ) : (
                    <div className={styles.emptyPrompt}>
                        <img
                            src={packImageUrl}
                            alt={`${setCode} booster`}
                            className={styles.heroPackImage}
                            onError={(e) => {
                                (e.currentTarget as HTMLImageElement).src = '/packs/default.png';
                            }}
                        />
                        <p className={styles.emptyText}>Ready to open? Hit <strong>"Open Booster"</strong> above to crack your pack!</p>
                    </div>
                )}
            </div>
        </div>
    );
}
