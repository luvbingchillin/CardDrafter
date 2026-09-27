import { type InputHTMLAttributes } from "react";
import styles from './Input.module.css'

interface InputProps extends
    InputHTMLAttributes<HTMLInputElement> {
    label?: string
    error?: string
}

export function Input({ label, error, className, ...props }: InputProps) {
    return (
        <div className={styles.inputWrapper}>
            {label && <label className={styles.label}>
                {label}</label>}
            <input
                className={`${styles.input} ${error ? styles.hasError : ''} ${className || ''}`}
                {...props}
            >
            </input>
            {error && <span className={styles.errorMessage}>{error}</span>}

        </div>
    )
}