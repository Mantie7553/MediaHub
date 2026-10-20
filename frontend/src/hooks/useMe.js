import useApi from "./useApi";

/**
 * The logged in user's profile
 * @returns me ({ id, username, email, role } or null until it loads), loading, error, and
 *  refresh to load it again in the background
 */
export default function useMe() {
    const { data: me, loading, error, reload } = useApi("/me");
    return { me, loading, error, refresh: () => reload(true) };
}
