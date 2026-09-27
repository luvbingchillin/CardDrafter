// src/api/auth.ts

export interface AuthSuccessResponse {
    user: {
        id: string;
        username: string;
        email: string;
    };
    message: string;
}

export interface AuthErrorResponse {
    error: string;
}

export async function loginUser(login: string, password: string) {
    const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ login, password }),
    });

    const data = await res.json();

    if (!res.ok) {
        // If HTTP status is 400 or 401, data will have { error: "..." }
        throw new Error((data as AuthErrorResponse).error || 'Failed to login');
    }

    return data as AuthSuccessResponse;
}
