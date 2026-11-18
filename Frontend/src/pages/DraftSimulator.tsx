import { useState, useEffect, useRef, useMemo } from "react";
import { getMe } from "../api/auth";
import type { Card } from "../api/pack";
import { CardUi } from "../components/CardUi";
import { Input } from "../components/ui/Input";
import styles from "./DraftSimulator.module.css";
import { copyToClipboard } from "../utils/deckExport";

export interface Seat {
  Index: number;
  UserID: string;
  IsReady: boolean;
  Client: object | null; // Non-null if player is currently connected
}

export type DraftState = 'LOBBY' | 'DRAFTING' | 'FINISHED';

export interface LobbyState {
  hostId: string;
  seats: Seat[];
  packSets: string[];
  state: DraftState;
}

export interface DraftUpdatePayload {
  round: number;
  totalRounds: number;
  currentPack: Card[] | null;
  draftedPool: Card[];
  queueCount: number;
  timeRemaining: number;
}

export interface TimerTickPayload {
  timeRemaining: number;
}

export interface WSMessage<T = unknown> {
  type: string;
  payload: T;
}

const CMC_BUCKETS = ['0–1', '2', '3', '4', '5', '6+', 'Lands'] as const;

function groupCardsByCmc(cards: Card[]): Record<string, Card[]> {
  const groups: Record<string, Card[]> = {
    '0–1': [],
    '2': [],
    '3': [],
    '4': [],
    '5': [],
    '6+': [],
    'Lands': [],
  };

  for (const card of cards) {
    const isLand = card.types?.some((t) => t.toLowerCase().includes('land'));
    if (isLand) {
      groups['Lands'].push(card);
      continue;
    }
    const cmc = card.mana_value ?? 0;
    if (cmc <= 1) groups['0–1'].push(card);
    else if (cmc === 2) groups['2'].push(card);
    else if (cmc === 3) groups['3'].push(card);
    else if (cmc === 4) groups['4'].push(card);
    else if (cmc === 5) groups['5'].push(card);
    else groups['6+'].push(card);
  }

  return groups;
}

// Helper to retrieve or generate a persistent guest identifier for unauthenticated players
function getOrCreateGuestId(): string {
  let guestId = sessionStorage.getItem('draft_guest_id');
  if (!guestId) {
    guestId = 'Guest_' + Math.random().toString(36).substring(2, 7).toUpperCase();
    sessionStorage.setItem('draft_guest_id', guestId);
  }
  return guestId;
}

export function DraftSimulator() {
  const [roomId, setRoomId] = useState('');
  const [roomIdInput, setRoomIdInput] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [lobbyState, setLobbyState] = useState<LobbyState | null>(null);
  const [draftState, setDraftState] = useState<DraftUpdatePayload | null>(null);
  const [currentUser, setCurrentUser] = useState<string>('');
  const [guestId] = useState(() => getOrCreateGuestId());
  const [selectedCardId, setSelectedCardId] = useState<string | null>(null);
  const [poolViewMode, setPoolViewMode] = useState<'curve' | 'grid'>('curve');
  const wsRef = useRef<WebSocket | null>(null);
  const [selectedSets, setSelectedSets] = useState<string[]>(['BLB', 'BLB', 'BLB']);
  const [timeRemaining, setTimeRemaining] = useState<number>(0);
  const [hasCopied, setHasCopied] = useState(false);

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/api/draft/${roomId}/ws`;
  const isHost = Boolean(lobbyState && currentUser && lobbyState.hostId === currentUser);

  // Group drafted cards by CMC for the Mana Curve deckbuilder view
  const groupedPool = useMemo(() => {
    return groupCardsByCmc(draftState?.draftedPool || []);
  }, [draftState?.draftedPool]);

  const creatureCount = useMemo(() => {
    return (draftState?.draftedPool || []).filter((c) =>
      c.types?.some((t) => t.toLowerCase().includes('creature'))
    ).length;
  }, [draftState?.draftedPool]);

  const nonCreatureCount = useMemo(() => {
    return (draftState?.draftedPool || []).filter(
      (c) =>
        !c.types?.some(
          (t) => t.toLowerCase().includes('creature') || t.toLowerCase().includes('land')
        )
    ).length;
  }, [draftState?.draftedPool]);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setRoomIdInput(event.target.value);
  };

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (roomIdInput.trim()) {
      setRoomId(roomIdInput.trim().toUpperCase());
    }
  };

  const handleLeaveRoom = () => {
    if (wsRef.current) {
      wsRef.current.close();
    }
    setRoomId('');
    setLobbyState(null);
    setDraftState(null);
    setSelectedCardId(null);
  };

  const sendAction = (type: string, payload: unknown = {}) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type, payload }));
    }
  };

  const handlePackCountChange = (count: number) => {
    setSelectedSets((prev) =>
      Array.from({ length: count }, (_, i) => prev[i] || 'BLB')
    );
  };

  const handleSetChange = (index: number, newSet: string) => {
    setSelectedSets((prev) => {
      const updated = [...prev];
      updated[index] = newSet;
      return updated;
    });
  };

  // Two-step pick model: First click highlights/selects; second click or Confirm button locks in
  const handleCardClick = (cardId: string) => {
    if (selectedCardId === cardId) {
      sendAction('PICK_CARD', { cardId });
      setSelectedCardId(null);
    } else {
      setSelectedCardId(cardId);
    }
  };

  const handleConfirmPick = () => {
    if (selectedCardId) {
      sendAction('PICK_CARD', { cardId: selectedCardId });
      setSelectedCardId(null);
    }
  };

  const handleCopyDeck = async () => {
    if (!draftState?.draftedPool) return;
    try {
      await copyToClipboard(draftState.draftedPool);
      setHasCopied(true);
      setTimeout(() => setHasCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy deck to clipboard:", err);
    }
  };

  useEffect(() => {
    if (!roomId) return;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => setIsConnected(true);
    ws.onclose = () => {
      setIsConnected(false);
    };
    ws.onerror = (err) => console.error("WebSocket error:", err);

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'LOBBY_UPDATE') {
          setLobbyState(msg.payload);
        } else if (msg.type === 'DRAFT_UPDATE') {
          setDraftState(msg.payload);
          setTimeRemaining(msg.payload.timeRemaining);
          setSelectedCardId(null); // Reset selection on new pack arrival
        } else if (msg.type === 'TIMER_TICK') {
          setTimeRemaining(msg.payload.timeRemaining);
        }
      } catch (err) {
        console.error("Failed to parse WebSocket message:", err);
      }
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [roomId, wsUrl]);

  useEffect(() => {
    const getUser = async () => {
      try {
        const user = await getMe();
        if (user) {
          setCurrentUser(user.id);
        }
      } catch (err) {
        if (err instanceof Error) {
          console.error("Failed to fetch current user:", err.message);
        }
      }
    };
    getUser();
  }, []);

  return (
    <div className={styles.container}>
      {/* 1. ROOM ENTRY SCREEN */}
      {!roomId && (
        <div className={styles.joinCard}>
          <h2 className={styles.joinTitle}>MTG Booster Draft</h2>
          <form onSubmit={handleSubmit} className={styles.joinForm}>
            <Input
              label="Room Code"
              placeholder="e.g. ALPHA, ROOM123"
              value={roomIdInput}
              onChange={handleChange}
              required
            />
            <button type="submit" className={styles.actionButton}>
              Join or Create Room
            </button>
          </form>
        </div>
      )}

      {/* 2. CONNECTING / LOADING SCREEN */}
      {roomId && !lobbyState && (
        <div className={styles.joinCard}>
          <div className={styles.waitingSpinner}>⏳</div>
          <h2 className={styles.joinTitle}>Connecting to Room {roomId}...</h2>
          <p className={styles.exportInfo}>
            {isConnected ? 'Waiting for lobby data from table host...' : 'Establishing real-time connection...'}
          </p>
          <div style={{ marginTop: '1rem', display: 'flex', flexDirection: 'column', gap: '0.5rem', alignItems: 'center' }}>
            <span className={styles.statusBadge}>Playing as: {currentUser ? currentUser : guestId + ' (Guest)'}</span>
            <button onClick={handleLeaveRoom} className={styles.btnLeave} style={{ marginTop: '0.75rem' }}>
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* 3. ACTIVE ROOM SCREEN */}
      {roomId && lobbyState && (
        <>
          {/* Top Bar Navigation & Status */}
          <header className={styles.topBar}>
            <div className={styles.titleArea}>
              <h1 className={styles.title}>Draft Room</h1>
              <span className={styles.roomTag}>{roomId}</span>
            </div>

            <div className={styles.topControls}>
              <div className={styles.statusBadge}>
                {isConnected ? '🟢 Connected' : '🟠 Reconnecting...'}
              </div>
              <button onClick={handleLeaveRoom} className={styles.btnLeave}>
                Leave Room
              </button>
            </div>
          </header>

          {/* PHASE A: LOBBY */}
          {lobbyState.state === 'LOBBY' && (
            <div className={styles.lobbyGrid}>
              {/* Seats & Players Table */}
              <section className={styles.lobbyCard}>
                <div className={styles.cardHeader}>
                  <span>Table Seats ({lobbyState.seats.length} Joined)</span>
                  <button
                    onClick={() => sendAction('TOGGLE_READY')}
                    className={styles.actionButton}
                  >
                    Toggle Ready
                  </button>
                </div>

                <div className={styles.seatsGrid}>
                  {lobbyState.seats.map((seat) => {
                    const isSelf = seat.UserID === currentUser;
                    return (
                      <div
                        key={seat.UserID}
                        className={`${styles.seatCard} ${isSelf ? styles.seatCardCurrent : ''}`}
                      >
                        <div className={styles.seatNumber}>Seat {seat.Index + 1}</div>
                        <div className={styles.seatUser}>
                          {seat.UserID} {isSelf && '(You)'}
                        </div>
                        <div className={styles.seatIndicators}>
                          <span>{seat.Client ? '🟢 Online' : '⚪ Offline'}</span>
                          <span>•</span>
                          <span>{seat.IsReady ? '✅ Ready' : '⏳ Waiting'}</span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </section>

              {/* Host Controls or Pack Roster */}
              <section className={styles.lobbyCard}>
                <div className={styles.cardHeader}>
                  <span>Pack Configuration</span>
                  {isHost && <span className={styles.statusBadge}>Host</span>}
                </div>

                {isHost ? (
                  <div className={styles.packConfigForm}>
                    <div className={styles.selectGroup}>
                      <label>Number of Rounds:</label>
                      <select
                        className={styles.selectInput}
                        value={selectedSets.length}
                        onChange={(e) => handlePackCountChange(Number(e.target.value))}
                      >
                        {[1, 2, 3, 4, 5].map((num) => (
                          <option key={num} value={num}>
                            {num} Packs
                          </option>
                        ))}
                      </select>
                    </div>

                    {selectedSets.map((setCode, index) => (
                      <div key={index} className={styles.selectGroup}>
                        <label>Pack {index + 1}:</label>
                        <select
                          className={styles.selectInput}
                          value={setCode}
                          onChange={(e) => handleSetChange(index, e.target.value)}
                        >
                          <option value="BLB">BLB (Bloomburrow)</option>
                          <option value="FDN">FDN (Foundations)</option>
                          <option value="WOE">WOE (Wilds of Eldraine)</option>
                          <option value="OTJ">OTJ (Outlaws of Thunder Junction)</option>
                          <option value="MH3">MH3 (Modern Horizons 3)</option>
                        </select>
                      </div>
                    ))}

                    <button
                      onClick={() => sendAction('SET_PACK_SETS', { packSets: selectedSets })}
                      className={styles.btnSecondary}
                    >
                      Apply Pack Setup
                    </button>

                    <button
                      onClick={() => sendAction('START_DRAFT')}
                      className={styles.actionButton}
                      disabled={
                        !lobbyState.packSets ||
                        lobbyState.packSets.length === 0 ||
                        lobbyState.seats.length === 0 ||
                        !lobbyState.seats.every((s) => !s.Client || s.IsReady)
                      }
                    >
                      Start Draft
                    </button>
                  </div>
                ) : (
                  <div>
                    {!lobbyState.packSets || lobbyState.packSets.length === 0 ? (
                      <p className={styles.packInstruction}>⏳ Waiting for host to select packs...</p>
                    ) : (
                      lobbyState.packSets.map((pack, index) => (
                        <div key={index} style={{ marginBottom: '0.5rem' }}>
                          <strong>Pack {index + 1}:</strong> {pack}
                        </div>
                      ))
                    )}
                  </div>
                )}
              </section>
            </div>
          )}

          {/* PHASE B: DRAFTING */}
          {lobbyState.state === 'DRAFTING' && (
            <>
              {/* Draft Status HUD */}
              <div className={styles.draftHud}>
                <div className={styles.hudGroup}>
                  <div className={styles.hudStat}>
                    <span className={styles.hudLabel}>Round</span>
                    <span className={styles.hudValue}>
                      {draftState?.round ?? 1} / {draftState?.totalRounds ?? lobbyState.packSets.length}
                    </span>
                  </div>
                  <div className={styles.hudStat}>
                    <span className={styles.hudLabel}>Time Left</span>
                    <span className={`${styles.hudValue} ${draftState?.currentPack && timeRemaining <= 5 ? styles.hudValueWarning : ""}`}>
                      {draftState?.currentPack ? `${timeRemaining}s` : "—"}
                    </span>
                  </div>
                  <div className={styles.hudStat}>
                    <span className={styles.hudLabel}>Direction</span>
                    <span className={styles.directionBadge}>
                      {((draftState?.round ?? 1) % 2 === 1) ? 'Passing Left ➡️' : 'Passing Right ⬅️'}
                    </span>
                  </div>
                </div>

                <div className={styles.hudGroup}>
                  {Boolean(draftState && draftState.queueCount > 0) && (
                    <span className={styles.queueBadge}>
                      ⚡ {draftState?.queueCount} Queued Pack{draftState && draftState.queueCount > 1 ? 's' : ''}
                    </span>
                  )}
                  <div className={styles.hudStat}>
                    <span className={styles.hudLabel}>Pool Count</span>
                    <span className={styles.hudValue}>{draftState?.draftedPool?.length ?? 0}</span>
                  </div>
                </div>
              </div>

              {/* Active Pack Cards */}
              <section className={styles.packSection}>
                <div className={styles.packHeader}>
                  <div>
                    <h2 className={styles.packTitle}>Current Pack</h2>
                    <span className={styles.packInstruction}>
                      {draftState?.currentPack && draftState.currentPack.length > 0
                        ? (selectedCardId
                            ? 'Click highlighted card again or press "Confirm Pick" to lock in'
                            : 'Click a card to select it')
                        : 'Waiting for neighbor'}
                    </span>
                  </div>

                  {selectedCardId && (
                    <button
                      onClick={handleConfirmPick}
                      className={styles.confirmPickButton}
                      aria-label="Confirm picked card"
                    >
                      Confirm Pick ✓
                    </button>
                  )}
                </div>

                {draftState?.currentPack && draftState.currentPack.length > 0 ? (
                  <div className={styles.cardGrid}>
                    {draftState.currentPack.map((card) => (
                      <CardUi
                        key={card.id || card.uuid}
                        card={card}
                        onClick={() => handleCardClick(card.id)}
                        isSelected={selectedCardId === card.id}
                      />
                    ))}
                  </div>
                ) : (
                  <div className={styles.waitingBox}>
                    <div className={styles.waitingSpinner} aria-hidden="true">⏳</div>
                    <div className={styles.waitingText}>
                      Waiting for your neighbor to pass the next pack...
                    </div>
                    {lobbyState?.seats && (
                      <div className={styles.neighborStatusList}>
                        {lobbyState.seats.map((seat) => (
                          <span
                            key={seat.UserID}
                            className={`${styles.neighborBadge} ${seat.UserID === currentUser ? styles.neighborBadgeSelf : ''}`}
                          >
                            Seat {seat.Index + 1}: {seat.UserID === currentUser ? 'You (Waiting)' : seat.UserID}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </section>

              {/* Player's Drafted Pool with Mana Curve View */}
              <section className={styles.poolSection}>
                <div className={styles.poolHeader}>
                  <div className={styles.poolTitleArea}>
                    <h3 className={styles.poolTitle}>
                      Your Drafted Pool ({draftState?.draftedPool?.length ?? 0} Cards)
                    </h3>
                    {draftState?.draftedPool && draftState.draftedPool.length > 0 && (
                      <span className={styles.poolSubtitle}>
                        {creatureCount} Creatures • {nonCreatureCount} Non-Creatures
                      </span>
                    )}
                  </div>

                  {draftState?.draftedPool && draftState.draftedPool.length > 0 && (
                    <div className={styles.viewModeToggle} role="group" aria-label="Pool view toggle">
                      <button
                        className={`${styles.toggleButton} ${poolViewMode === 'curve' ? styles.toggleActive : ''}`}
                        onClick={() => setPoolViewMode('curve')}
                        type="button"
                      >
                        📊 Mana Curve
                      </button>
                      <button
                        className={`${styles.toggleButton} ${poolViewMode === 'grid' ? styles.toggleActive : ''}`}
                        onClick={() => setPoolViewMode('grid')}
                        type="button"
                      >
                        🎴 Flat Grid
                      </button>
                    </div>
                  )}
                </div>

                {draftState?.draftedPool && draftState.draftedPool.length > 0 ? (
                  poolViewMode === 'curve' ? (
                    <div className={styles.curveContainer}>
                      {CMC_BUCKETS.map((bucket) => {
                        const bucketCards = groupedPool[bucket] || [];
                        return (
                          <div key={bucket} className={styles.curveColumn}>
                            <div className={styles.curveHeader}>
                              <span className={styles.curveCostBadge}>CMC {bucket}</span>
                              <span className={styles.curveCount}>{bucketCards.length}</span>
                            </div>
                            <div className={styles.curveCardStack}>
                              {bucketCards.map((card, idx) => (
                                <div key={`${card.id}-${idx}`} className={styles.stackedCard}>
                                  <CardUi card={card} />
                                </div>
                              ))}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <div className={styles.poolGrid}>
                      {draftState.draftedPool.map((card, idx) => (
                        <div key={`${card.id}-${idx}`}>
                          <CardUi card={card} />
                        </div>
                      ))}
                    </div>
                  )
                ) : (
                  <p className={styles.packInstruction}>
                    No cards drafted yet. Pick your first card above!
                  </p>
                )}
              </section>
            </>
          )}

          {/* PHASE C: FINISHED */}
          {lobbyState.state === 'FINISHED' && (
            <section className={styles.poolSection}>
              <div className={styles.poolHeader}>
                <div>
                  <h2>🎉 Draft Complete!</h2>
                  <p style={{ color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
                    You drafted {draftState?.draftedPool?.length ?? 0} cards ({creatureCount} Creatures, {nonCreatureCount} Non-Creatures).
                  </p>
                </div>
                <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
                  {hasCopied && <span className={styles.copySuccessBadge}>Copied to clipboard!</span>}
                  <button
                    onClick={handleCopyDeck}
                    className={styles.actionButton}
                    aria-label="Export deck for MTG Arena"
                  >
                    Export for MTG Arena
                  </button>
                </div>
              </div>

              {draftState?.draftedPool && draftState.draftedPool.length > 0 && (
                <div className={styles.curveContainer} style={{ marginTop: '1rem' }}>
                  {CMC_BUCKETS.map((bucket) => {
                    const bucketCards = groupedPool[bucket] || [];
                    return (
                      <div key={bucket} className={styles.curveColumn}>
                        <div className={styles.curveHeader}>
                          <span className={styles.curveCostBadge}>CMC {bucket}</span>
                          <span className={styles.curveCount}>{bucketCards.length}</span>
                        </div>
                        <div className={styles.curveCardStack}>
                          {bucketCards.map((card, idx) => (
                            <div key={`${card.id}-${idx}`} className={styles.stackedCard}>
                              <CardUi card={card} />
                            </div>
                          ))}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </section>
          )}
        </>
      )}
    </div>
  );
}
