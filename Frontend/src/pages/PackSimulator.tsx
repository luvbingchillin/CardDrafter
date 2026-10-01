import { useState, useEffect } from 'react';
import { getCardPacks, type Pack } from '../api/pack';
import styles from './PackSimulator.module.css';

export function PackSimulator() {
  const [packData, setPackData] = useState<Pack[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
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
          setError('unknown error occured');
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
      <div className={styles.header}>
        <h1 className={styles.title}>Booster Simulator</h1>
        <p className={styles.subtitle}>Select an expansion pack to open</p>
      </div>
      <div className={styles.packGrid}>
        {packData.map((pack) => (
          <div key={pack.setCode} className={styles.packCard}>
            <div className={styles.imageWrapper}>
              <img
                src={pack.image}
                alt={pack.name}
                className={styles.packImage}
              />
            </div>
            <div className={styles.packInfo}>
              <span className={styles.setCode}>{pack.setCode}</span>
              <p className={styles.packName}>{pack.name}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
