import { useState, useEffect } from 'react';
import { Route, Routes, NavLink } from 'react-router-dom';
import { useSearchParams } from 'react-router-dom';
import { LoginModal } from './components/LoginModal';
import { PackSimulator } from './pages/PackSimulator';
import { DraftSimulator } from './pages/DraftSimulator';
import { PackOpeningPage } from './pages/PackOpeningPage';
import { getMe } from './api/auth';
import './App.css';

function App() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [isAuth, setIsAuth] = useState(() => searchParams.get('auth') === 'success');
  const [loginPopup, SetLoginPopup] = useState(false);

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
        <NavLink to="/" className="brand">
          <span className="brand-text">Placeholder</span>
        </NavLink>
        <nav className="nav">
          <NavLink className="navLink" to="/packs">
            Pack simulator
          </NavLink>
          <NavLink className="navLink" to="/draft">
            Draft simulator
          </NavLink>
        </nav>
        <div className="userSection">
          {!isAuth ? (
            <button className="loginBtn" onClick={() => SetLoginPopup(true)}>
              <div>Login/SignUp</div>
            </button>
          ) : (
            <div className="userBadge">
              <span>My account</span>
            </div>
          )}
        </div>
      </header>

      <LoginModal
        isOpen={loginPopup}
        onClose={() => {
          SetLoginPopup(false);
        }}
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
