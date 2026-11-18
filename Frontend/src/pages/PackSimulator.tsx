import { useState, useEffect } from 'react';
import { getCardPacks, type Pack } from '../api/pack';
import { PackUi } from '../components/PackUi';
import styles from './PackSimulator.module.css';
import { useNavigate } from 'react-router-dom';

export function PackSimulator() {
  const [packData, setPackData] = useState<Pack[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    const fetchPacks = async () => {
      try {
        setLoading(true);
        const result = await getCardPacks();
        setPackData(result);
      } catch (error) {
        if (error instanceof Error) {
          setError(error.message);
        } else {
          setError('Unknown error occurred');
        }
      } finally {
        setLoading(false);
      }
    };
    fetchPacks();
  }, []);

  if (loading) {
    return <div className={styles.statusMessage}>Loading booster packs...</div>;
  }

  if (error) {
    return <div className={styles.errorMessage}>{error}</div>;
  }

  return (
    <div className={styles.pageContainer}>
      <header className={styles.header}>
        <h1 className={styles.title}>Booster Packs</h1>
        <p className={styles.subtitle}>Select a set to crack packs and test pull probabilities</p>
      </header>
      <main className={styles.packGrid}>
        {packData.map((pack) => (
          <PackUi pack={pack} key={pack.setCode} onClick={() => navigate(`/packs/${pack.setCode}`)} />
        ))}
      </main>
    </div>
  );
}
