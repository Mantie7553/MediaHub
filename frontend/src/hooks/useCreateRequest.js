import { useState } from 'react'
import api from '../services/api'
import { errorMessage } from '../utils/errors'

export default function useCreateRequest(id) {
    const [requesting, setRequesting] = useState(false);
    const [requestMsg, setRequestMsg] = useState("");

    function createRequest() {
        setRequesting(true);
        setRequestMsg("");
        api.post("/requests", { media_item_id: id })
        .then(() => setRequestMsg("Download requested!"))
        .catch(err => setRequestMsg(errorMessage(err, "Could not request the download")))
        .finally(() => setRequesting(false))
    }

    return { requesting, requestMsg, createRequest }
}
