import { useParams, useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { type Card, type SetAnalytics, openPack, getSetAnalytics } from '../api/pack';
import { CardUi } from '../components/CardUi';
import styles from './PackOpeningPage.module.css'

export function PackOpeningPage() {
    const { setCode } = useParams<{ setCode: string }>();
    const navigate = useNavigate()
    const [cards, setCards] = useState<Card[]>([])
    const [analytics, setAnalytics] = useState<SetAnalytics | null>(null)
    const [error, setError] = useState('')
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        if (!setCode) return
        const getSetAnalyticFunc = async () => {
            try {
                const response = await getSetAnalytics(setCode)
                setAnalytics(response)
            } catch (error) {
                console.error(error)
            }

        }
        getSetAnalyticFunc()

    }, [setCode])

    const openPackBySet = async () => {
        if (!setCode || loading) return;
        try {
            setLoading(true);
            setError('');
            const openedCards = await openPack(setCode)
            setCards(openedCards)
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

        } catch (error) {
            if (error instanceof Error) {
                setError(error.message);
            } else {
                setError('Failed to open pack');
            }
        } finally {
            setLoading(false)
        }


    }
    const packImageUrl = `/packs/${setCode?.toUpperCase()}.png`;
    const hasOpened = cards.length > 0;
    return (
        <>
            <div>
                <button></button>
            </div>
            {
                hasOpened && (
                    <div>
                        {cards.map((card, index) => (
                            <CardUi key={card.id} card={card}></CardUi>
                        ))}
                    </div>
                )
            }
        </>
    )
}
