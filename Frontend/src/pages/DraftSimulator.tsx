import { useState, useEffect, useRef } from "react";
import { getMe } from "../api/auth";
import type { Card } from "../api/pack";
import { CardUi } from "../components/CardUi";
import { Input } from "../components/ui/Input";
import styles from "./DraftSimulator.module.css";

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
}

export interface WSMessage<T = unknown> {
  type: string;
  payload: T;
}

export function DraftSimulator() {
  const [roomId, setRoomId] = useState('');
  const [roomIdInput, setRoomIdInput] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [lobbyState, setLobbyState] = useState<LobbyState | null>(null);
  const [draftState, setDraftState] = useState<DraftUpdatePayload | null>(null);
  const [currentUser, setCurrentUser] = useState<string>('');
  const wsRef = useRef<WebSocket | null>(null);
  const [selectedSets, setSelectedSets] = useState<string[]>(['BLB', 'BLB', 'BLB']);

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/api/draft/${roomId}/ws`;
  const isHost = Boolean(lobbyState && currentUser && lobbyState.hostId === currentUser);

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

  const handlePickCard = (cardId: string) => {
    sendAction('PICK_CARD', { cardId });
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

      {/* 2. ACTIVE ROOM SCREEN */}
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
                  <h2 className={styles.packTitle}>Current Pack</h2>
                  <span className={styles.packInstruction}>
                    {draftState?.currentPack && draftState.currentPack.length > 0
                      ? 'Click 1 card to draft and pass the rest'
                      : 'Waiting for neighbor'}
                  </span>
                </div>

                {draftState?.currentPack && draftState.currentPack.length > 0 ? (
                  <div className={styles.cardGrid}>
                    {draftState.currentPack.map((card) => (
                      <div
                        key={card.id || card.uuid}
                        onClick={() => handlePickCard(card.id)}
                        className={styles.cardPickable}
                        title={`Pick ${card.name}`}
                      >
                        <CardUi card={card} />
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className={styles.waitingBox}>
                    <div className={styles.waitingSpinner}>⏳</div>
                    <div className={styles.waitingText}>
                      Waiting for the player next to you to pass their pack...
                    </div>
                  </div>
                )}
              </section>

              {/* Player's Drafted Pool */}
              <section className={styles.poolSection}>
                <div className={styles.poolHeader}>
                  <h3>Your Drafted Pool ({draftState?.draftedPool?.length ?? 0} Cards)</h3>
                </div>

                {draftState?.draftedPool && draftState.draftedPool.length > 0 ? (
                  <div className={styles.poolGrid}>
                    {draftState.draftedPool.map((card, idx) => (
                      <div key={`${card.id}-${idx}`}>
                        <CardUi card={card} />
                      </div>
                    ))}
                  </div>
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
              <h2>🎉 Draft Complete!</h2>
              <p>You drafted {draftState?.draftedPool?.length ?? 0} cards.</p>
              <div className={styles.poolGrid}>
                {draftState?.draftedPool?.map((card, idx) => (
                  <div key={`${card.id}-${idx}`}>
                    <CardUi card={card} />
                  </div>
                ))}
              </div>
            </section>
          )}
        </>
      )}
    </div>
  );
}
