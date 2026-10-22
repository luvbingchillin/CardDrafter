import { useState, useEffect, useRef } from "react";
import { getMe } from "../api/auth";
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

export interface WSMessage<T = unknown> {
  type: string;
  payload: T;
}




export function DraftSimulator() {
  const [roomId, setRoomId] = useState('')
  const [roomIdInput, setRoomIdInput] = useState('')
  const [isConnected, setIsConnected] = useState(false)
  const [lobbyState, setLobbyState] = useState<LobbyState | null>(null)
  const [currentUser, setCurrentUser] = useState<string>('')
  const wsRef = useRef<WebSocket | null>(null);
  const [selectedSets, setSelectedSets] = useState<string[]>(['BLB', 'BLB', 'BLB']);

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/api/draft/${roomId}/ws`;
  const isHost = Boolean(lobbyState && currentUser && lobbyState.hostId === currentUser);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const curRoomIdInput = event.target.value
    setRoomIdInput(curRoomIdInput)
  }

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setRoomId(roomIdInput)
  }
  const handleLeaveRoom = () => {
    setRoomId('')
    setLobbyState(null)
  }

  const sendAction = (type: string, payload: unknown = {}) => {
    if (wsRef.current?.readyState == WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type, payload }))
    }
  }

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

  useEffect(() => {
    if (!roomId) return;
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws
    ws.onopen = () => setIsConnected(true);
    ws.onclose = () => setIsConnected(false);
    ws.onerror = (err) => console.error(err);
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)
      if (msg.type === 'LOBBY_UPDATE') {
        setLobbyState(msg.payload)
      }
    }
    return () => {
      ws.close()
      wsRef.current = null
    }
  }, [roomId])

  useEffect(() => {
    const getUser = async () => {
      const user = await getMe()
      if (user) {
        setCurrentUser(user.id)
      }
    }
    getUser()
  }, [])

  return (
    <>

      <div>DraftSim</div>
      {!roomId && <form onSubmit={handleSubmit}>
        <div>
          <label>Room ID:</label>
          <input value={roomIdInput} onChange={handleChange}></input>
        </div>
        <button type="submit">Join Room</button>
      </form>}
      {roomId && lobbyState && <div>
        {isHost ? (
          <div>
            <h3>Draft Pack Configuration (Host)</h3>
            <div>
              <label>Number of Packs: </label>
              <select
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

            <div>
              {selectedSets.map((setCode, index) => (
                <div key={index}>
                  <label>Pack {index + 1}: </label>
                  <select
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
            </div>

            <button onClick={() => sendAction('SET_PACK_SETS', { packSets: selectedSets })}>
              Update Packs
            </button>
            <button
              onClick={() => sendAction('START_DRAFT')}
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
            <h3>Draft Packs Selected by Host:</h3>
            {!lobbyState.packSets || lobbyState.packSets.length === 0 ? (
              <div>⏳ Waiting for host to select packs...</div>
            ) : (
              lobbyState.packSets.map((pack, index) => (
                <div key={index}>
                  Pack {index + 1}: {pack}
                </div>
              ))
            )}
          </div>
        )}
        <div>{roomId}</div>
        <div>{lobbyState.state}</div>
        <div>Connection badge</div>
        <div>{isConnected ? '🟢 Connected' : '🟠 Connecting...'}</div>

        <button onClick={handleLeaveRoom}>Leave Room</button>
        {lobbyState.seats.map((seat) => {
          return (
            <div key={seat.UserID}>
              <div>Seat index:{seat.Index + 1}</div>
              <div>User: {seat.UserID}</div>
              <div>Status</div>
              <div>Status: {seat.IsReady ? '✅ Ready' : '⏳ Not Ready'}</div>
              <div>Online</div>
              <div>{seat.Client ? '🟢 Online' : '⚪ Offline'}</div>


            </div>
          )
        })}
        <button onClick={() => sendAction('TOGGLE_READY')}>
          Ready up
        </button>
      </div>}

    </>
  )

}
