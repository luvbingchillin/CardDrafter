import { useState, useEffect } from 'react';
import { Route, Routes, NavLink } from 'react-router-dom';
import { useSearchParams } from 'react-router-dom';
import { LoginModal } from './components/LoginModal';
import { PackSimulator } from './pages/PackSimulator';
import { DraftSimulator } from './pages/DraftSimulator';
import { PackOpeningPage } from './pages/PackOpeningPage';
import './App.css';
function App() {
  const [isAuth, setIsAuth] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();

  const [loginPopup, SetLoginPopup] = useState(false);

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
