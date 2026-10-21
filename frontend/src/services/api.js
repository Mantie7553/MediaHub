import axios from 'axios'
import { forgetSession, getAccessToken, refreshSession } from './session'

const BASE_URL = import.meta.env.VITE_API_URL

// the endpoints that sign in or out with the session cookie, which a refresh can't help
const sessionEndpoints = ["/auth/login", "/auth/register", "/auth/refresh", "/auth/logout"];

const api = axios.create({
  baseURL: BASE_URL,
})

api.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
    (response) => response,
    async (error) => {
        const original = error.config;

        // the access token expired (they last 15 minutes): get a new one from the session cookie
        // and try once more. Requests failing together share one refresh
        // a 401 from signing in is a wrong password, not an expired session. Other /auth
        // routes (changing the password) use the access token like any request
        const authCall = sessionEndpoints.includes(original?.url);
        if (error.response?.status === 401 && original && !original._retry && !authCall) {
            original._retry = true;
            const token = await refreshSession();
            if (token) {
                original.headers.Authorization = `Bearer ${token}`;
                return api(original);
            }
            forgetSession();
            window.location.href = "/login";
        }

        return Promise.reject(error);
    }
)

export default api
