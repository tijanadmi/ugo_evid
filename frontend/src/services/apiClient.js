const baseURL = (import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/$/, '');

export class ApiError extends Error {
  constructor(message, status) { super(message); this.status = status; }
}

export async function request(path, { token, body, signal, method = 'GET' } = {}) {
  let response;
  try {
    response = await fetch(`${baseURL}${path}`, {
      method, signal,
      headers: {
        ...(body ? { 'Content-Type': 'application/json' } : {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      ...(body ? { body: JSON.stringify(body) } : {}),
    });
  } catch (error) {
    if (error.name === 'AbortError') throw error;
    throw new ApiError('Veza sa serverom nije dostupna. Pokušajte ponovo.', 0);
  }
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    throw new ApiError(data?.error || 'Zahtev nije uspeo. Pokušajte ponovo.', response.status);
  }
  if (!data) throw new ApiError('Server je vratio neispravan odgovor.', response.status);
  return data;
}
