import useApi from "./useApi";

/**
 * The logged in user's profile
 * @returns me ({ id, username, email, role } or null until it loads), loading and error
 */
export default function useMe() {
    const { data: me, loading, error } = useApi("/me");
    return { me, loading, error };
}
