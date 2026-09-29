import { useState, useEffect } from 'react'
import { Link, Route, Routes } from 'react-router-dom'
import { useSearchParams } from 'react-router-dom';
import { LoginModal } from './components/LoginModal'
import { PackSimulator } from './pages/PackSimulator'
import { DraftSimulator } from './pages/DraftSimulator'
function App() {
  const [isAuth, setIsAuth] = useState(false)
  const [searchParams, setSearchParams] = useSearchParams();

  const [loginPopup, SetLoginPopup] = useState(false)

  useEffect(() => {
    // If returning from Google OAuth redirect:
    if (searchParams.get('auth') === 'success') {
      setIsAuth(true);
      // Clean up the URL by removing "?auth=success"
      setSearchParams({});
    }
  }, [searchParams]);
  return (
    <>
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
    </>
  )
}

export default App
