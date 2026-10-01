import { useState } from 'react';
import { Input } from './ui/Input';
import styles from './LoginModal.module.css';
import { loginUser, registerUser } from '../api/auth';

interface LoginModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function LoginModal({ isOpen, onClose, onSuccess }: LoginModalProps) {
  const [user, SetUser] = useState('');
  const [password, setPassword] = useState('');
  const [email, setEmail] = useState('');
  const [errorText, setErrorText] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isLogin, setIsLogin] = useState(true);
  const handleSubmitLogin = async (e) => {
    e.preventDefault();
    setErrorText('');
    setIsLoading(true);
    try {
      await loginUser(user, password);
      onSuccess();
      onClose();
    } catch (err) {
      setErrorText(err.message || 'Network error');
    } finally {
      setIsLoading(false);
    }
  };
  const handleSubmitRegister = async (e) => {
    e.preventDefault();
    setErrorText('');
    setIsLoading(true);
    try {
      await registerUser(user, email, password);
      onSuccess();
      onClose();
    } catch (err) {
      setErrorText(err.message || 'Network error');
    } finally {
      setIsLoading(false);
    }
  };
  const switchMode = (loginMode: boolean) => {
    setIsLogin(loginMode);
    setErrorText('');
    SetUser('');
    setPassword('');
    setEmail('');
  };

  if (!isOpen) return null;
  return (
    <div className={styles.overlay} onClick={onClose}>
      <div
        className={`${styles.modal} ${isLogin ? styles.modalLogin : styles.modalRegister}`}
        onClick={(e) => e.stopPropagation()}
      >
        {isLogin ? (
          <>
            <div className={styles.header}>
              <h3>Login</h3>
              <button className={styles.closeButton} onClick={onClose}>
                <CloseIcon />
              </button>
            </div>
            {errorText && <div className={styles.errorAlert}>{errorText}</div>}
            <form
              key="login"
              onSubmit={handleSubmitLogin}
              className={styles.form}
            >
              <Input
                placeholder="Userame/Email"
                type="text"
                label="Username or Email"
                value={user}
                onChange={(e) => {
                  SetUser(e.target.value);
                }}
              />
              <Input
                placeholder="Password"
                type="password"
                label="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <button className={styles.submit} type="submit">
                Submit
              </button>
              <div className={styles.divider}>
                <span>or continue with</span>
              </div>
              <div className={styles.socialRow}>
                <button
                  onClick={() => {
                    window.location.href =
                      'http://localhost:8000/api/auth/google';
                  }}
                  type="button"
                  className={styles.iconOnlyButton}
                >
                  <GoogleIcon />
                </button>
              </div>
              <p className={styles.footerText}>
                Don't have an account?{' '}
                <button
                  className={styles.toggle}
                  type="button"
                  onClick={() => {
                    switchMode(false);
                  }}
                >
                  Sign Up
                </button>
              </p>
            </form>
          </>
        ) : (
          <>
            <div className={styles.header}>
              <h3>Register</h3>
              <button className={styles.closeButton} onClick={onClose}>
                <CloseIcon />
              </button>
            </div>
            {errorText && <div className={styles.errorAlert}>{errorText}</div>}
            <form
              key="register"
              onSubmit={handleSubmitRegister}
              className={styles.form}
            >
              <Input
                placeholder="Userame"
                type="text"
                label="Username"
                value={user}
                onChange={(e) => {
                  SetUser(e.target.value);
                }}
              />
              <Input
                placeholder="Email"
                type="text"
                label="Email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                }}
              />
              <Input
                placeholder="Password"
                type="password"
                label="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <button className={styles.submit} type="submit">
                {isLoading ? 'Creating account...' : 'Submit'}
              </button>
              <div className={styles.divider}>
                <span>or continue with</span>
              </div>
              <div className={styles.socialRow}>
                <button
                  onClick={() => {
                    window.location.href =
                      'http://localhost:8000/api/auth/google';
                  }}
                  type="button"
                  className={styles.iconOnlyButton}
                >
                  <GoogleIcon />
                </button>
              </div>
              <p className={styles.footerText}>
                Already have an account?{' '}
                <button
                  className={styles.toggle}
                  type="button"
                  onClick={() => {
                    switchMode(true);
                  }}
                >
                  Log In
                </button>
              </p>
            </form>
          </>
        )}
      </div>
    </div>
  );
}

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24">
      <path
        fill="#4285F4"
        d="M23.745 12.27c0-.7-.06-1.4-.19-2.07H12v4.51h6.6c-.29 1.52-1.14 2.82-2.4 3.68v3.05h3.88c2.27-2.09 3.66-5.17 3.66-9.17z"
      />
      <path
        fill="#34A853"
        d="M12 24c3.24 0 5.95-1.08 7.93-2.91l-3.88-3.05c-1.08.72-2.45 1.16-4.05 1.16-3.12 0-5.77-2.1-6.72-4.93H1.25v3.15C3.26 21.36 7.36 24 12 24z"
      />
      <path
        fill="#FBBC05"
        d="M5.28 14.27c-.25-.72-.38-1.49-.38-2.27s.13-1.55.38-2.27V6.58H1.25C.45 8.18 0 10.03 0 12s.45 3.82 1.25 5.42l4.03-3.15z"
      />
      <path
        fill="#EA4335"
        d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C17.95 1.19 15.24 0 12 0 7.36 0 3.26 2.64 1.25 6.58l4.03 3.15c.95-2.83 3.6-4.98 6.72-4.98z"
      />
    </svg>
  );
}

function CloseIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
      <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z" />
    </svg>
  );
}
