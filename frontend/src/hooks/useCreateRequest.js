import { useState } from 'react'
import api from '../services/api'

export default function useCreateRequest(id) {
    const [requesting, setRequesting] = useState(false);
    const [requestMsg, setRequestMsg] = useState("");

    function createRequest() {
        setRequesting(true);
        setRequestMsg("");
        api.post("/requests", { media_item_id: id })
        .then(() => setRequestMsg("Download requested!"))
        .catch(err => setRequestMsg(err.response?.data?.error ?? err.message))
        .finally(() => setRequesting(false))
    }

    return { requesting, requestMsg, createRequest }
}