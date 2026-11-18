import { useState, useEffect } from 'react';
import { Route, Routes, NavLink } from 'react-router-dom';
import { useSearchParams } from 'react-router-dom';
import { LoginModal } from './components/LoginModal';
import { PackSimulator } from './pages/PackSimulator';
import { DraftSimulator } from './pages/DraftSimulator';
import { PackOpeningPage } from './pages/PackOpeningPage';
import { LogoIcon } from './components/ui/Icons';
import { getMe } from './api/auth';
import './App.css';

function App() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [isAuth, setIsAuth] = useState(() => searchParams.get('auth') === 'success');
  const [loginPopup, setLoginPopup] = useState(false);

  useEffect(() => {
    if (searchParams.get('auth') === 'success') {
      setSearchParams({}, { replace: true });
    }
    getMe().then((user) => {
      if (user) {
        setIsAuth(true);
      }
    });
  }, [searchParams, setSearchParams]);

  return (
    <>
      <header className="header">
        <NavLink to="/" className="brand" aria-label="CardSim Home">
          <LogoIcon size={22} className="brandIcon" />
          <span className="brand-text">CardSim</span>
        </NavLink>
        <nav className="nav" aria-label="Primary navigation">
          <NavLink className="navLink" to="/packs">
            Pack Simulator
          </NavLink>
          <NavLink className="navLink" to="/draft">
            Draft Simulator
          </NavLink>
        </nav>
        <div className="userSection">
          {!isAuth ? (
            <button
              className="loginBtn"
              onClick={() => setLoginPopup(true)}
              aria-label="Open sign in dialog"
            >
              Sign In
            </button>
          ) : (
            <div className="userBadge" role="status">
              <span>My Account</span>
            </div>
          )}
        </div>
      </header>

      <LoginModal
        isOpen={loginPopup}
        onClose={() => setLoginPopup(false)}
        onSuccess={() => setIsAuth(true)}
      />

      <Routes>
        <Route path="/" element={<PackSimulator />} />
        <Route path="/packs" element={<PackSimulator />} />
        <Route path="/packs/:setCode" element={<PackOpeningPage />} />
        <Route path="/draft" element={<DraftSimulator />} />
      </Routes>
    </>
  );
}

export default App;
