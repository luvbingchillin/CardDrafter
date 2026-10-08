import { useParams } from 'react-router-dom';
import { useState } from 'react';
import { type Card, openPack } from '../api/pack';
import { CardUi } from '../components/CardUi';

export function PackOpeningPage() {
    const { setCode } = useParams<{ setCode: string }>();
    const [cards, setCards] = useState<Card[]>([])
    const [error, setError] = useState('')
    const [loading, setLoading] = useState('')
    const openPackBySet = async () => {
        try {
            const openedCards = await openPack(setCode)
            setCards(openedCards)

        } catch (error) {
            if (error instanceof Error) {
                setError(error.message);
            } else {
                setError('Failed to open pack');
            }
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
