import { useState } from 'react'
import { BrowserRouter, Link, Route, Routes } from 'react-router-dom'
import { LoginModal } from './components/LoginModal'
import { PackSimulator } from './pages/PackSimulator'
import { DraftSimulator } from './pages/DraftSimulator'
function App() {
  const [isAuth, setIsAuth] = useState(false)
  const [loginPopup, SetLoginPopup] = useState(false)
  return (
    <BrowserRouter>
      <h1>Card Simulator</h1>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <nav>
          <Link
            to="/packs"
          >
            Pack simulator
          </Link>
          <Link
            to="/draft"
          >
            Draft simulator
          </Link>
        </nav>
        <button onClick={() => SetLoginPopup(true)}
        >
          //user icon here
          <div>Login/SignUp</div>
        </button>
      </header>
      <LoginModal
        isOpen={loginPopup}
        onClose={() => { SetLoginPopup(false) }}
        onSuccess={() => setIsAuth(true)}
      />

      <Routes>
        <Route path="/packs" element={<PackSimulator />} />
        <Route path="/draft" element={<DraftSimulator />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
