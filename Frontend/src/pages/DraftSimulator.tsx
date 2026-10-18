import { useState, useEffect, useRef } from "react";
import { Form } from "react-router-dom";

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
  const wsRef = useRef<WebSocket | null>(null);
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/api/draft/${roomId}/ws`;

  const handleChange = (event) => {
    const curRoomIdInput = event.target.value
    setRoomIdInput(curRoomIdInput)
  }

  const handleSubmit = (e) => {
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
