import { useState } from "react";
import styles from "./LoginModal.module.css"

interface LoginModalProps {
    isOpen: boolean
    onClose: () => void
    onSuccess: () => void
}


export function LoginModal({ isOpen, onClose, onSuccess }: LoginModalProps) {
    const [user, SetUser] = useState('')
    const [password, setPassword] = useState('')
    const [errorText, SetErrorText] = useState('')
    const handleSubmit = async (e) => {
        try {
            /*             e.preventDefault()
                        let response = await submitLogin(user, password)
                        if (response.ok == true) {
                            onSuccess()
                            onClose()
                        } else {
                            SetErrorText(`Error:${response.error}`)
                        } */
        } catch (err) {
            SetErrorText('Network error')
        }

    }
    if (!isOpen) return null
    return (
        <div className={styles.overlay} onClick={onClose}>
            <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
                <div className={styles.header}>
                    <h3>Login/Sign Up</h3>
                    <button className={styles.closeButton} onClick={onClose}>X</button>
                    {errorText &&
                        <div style={{ color: "red" }}>{errorText}</div>
                    }
                </div>
                <form onSubmit={handleSubmit} className={styles.form}>
                    <input
                        placeholder='Userame/Email' type="text"
                        value={user} onChange={(e) => { SetUser(e.target.value) }}
                    />
                    <input
                        placeholder='Password' type="password"
                        value={password} onChange={(e) => setPassword(e.target.value)}
                    />
                    <button type='submit'>
                        Submit
                    </button>
                    <div>
                        <div>
                            Or Sign in Via:
                        </div>
                        <button
                            onClick={() => { window.location.href = "temp/auth/googe" }}
                            type="button"
                        >
                            Sign in with google
                        </button>
                    </div>
                </form>
            </div>
        </div>
    )
}
