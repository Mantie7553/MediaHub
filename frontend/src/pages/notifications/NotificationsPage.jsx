import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { BellOff, CheckCheck, Circle, CircleCheck } from "lucide-react"
import PageHeader from "../../components/layout/PageHeader"
import SegmentedTabs from "../../components/SegmentedTabs"
import EmptyState from "../../components/states/EmptyState"
import ErrorState from "../../components/states/ErrorState"
import { ListSkeleton } from "../../components/states/Skeletons"
import useApi from "../../hooks/useApi"
import { markAllNotificationsRead, markNotificationRead } from "../../services/notificationsService"
import useNotificationStore from "../../stores/useNotificationStore"
import { toast } from "../../stores/useToastStore"
import Format from "../../utils/format"
import { notificationKind, visibleNotifications } from "../../utils/notifications"
import { readStored, writeStored } from "../../utils/storage"

const VIEW_KEY = "notifications_view"
const views = [
    { key: "unread", label: "Unread" },
    { key: "all", label: "All" },
]

/**
 * The user's notifications, unread first. Opening one marks it read and goes where it needs
 * the user; read ones can be hidden
 */
export default function NotificationsPage() {
    const navigate = useNavigate()
    const [view, setView] = useState(() => readStored(VIEW_KEY, "unread", ["unread", "all"]))
    const { data, loading, error, reload } = useApi("/me/notifications", { initial: [], poll: 30000 })
    const refreshCount = useNotificationStore(state => state.refresh)
    // read states changed here before the list reloads, so the page answers at once
    const [changed, setChanged] = useState({})

    const items = data.map(n => n.id in changed ? { ...n, read_at: changed[n.id] } : n)
    const shown = visibleNotifications(items, view)
    const unread = items.filter(n => !n.read_at).length

    function pickView(key) {
        setView(key)
        writeStored(VIEW_KEY, key)
    }

    function setRead(item, read) {
        setChanged(prev => ({ ...prev, [item.id]: read ? new Date().toISOString() : null }))
        return markNotificationRead(item.id, read)
            .then(() => { refreshCount(); reload(true) })
            .catch(err => {
                setChanged(prev => ({ ...prev, [item.id]: item.read_at }))
                toast.error(err, `Could not mark it ${read ? "read" : "unread"}`)
            })
    }

    function open(item) {
        if (!item.read_at) setRead(item, true)
        if (item.link) navigate(item.link)
    }

    function readAll() {
        const now = new Date().toISOString()
        setChanged(Object.fromEntries(items.filter(n => !n.read_at).map(n => [n.id, now])))
        markAllNotificationsRead()
            .then(() => { refreshCount(); reload(true) })
            .catch(err => { setChanged({}); toast.error(err, "Could not mark them read") })
    }

    function body() {
        if (loading) return <ListSkeleton rows={6}/>
        if (error) return <ErrorState error={error} onRetry={() => reload()}/>
        if (shown.length === 0) return view === "unread" && items.length > 0
            ? <EmptyState icon={CheckCheck} title="You're all caught up" hint="Nothing unread. Switch to All to see older notifications."
                action={{ label: "Show all", onClick: () => pickView("all") }}/>
            : <EmptyState icon={BellOff} title="No notifications yet" hint="Requests to approve, failed downloads and new downloads show up here."/>
        return <ul className="flex flex-col gap-2">
            {shown.map(item => <NotificationRow key={item.id} item={item} onOpen={open} onToggleRead={setRead}/>)}
        </ul>
    }

    return <div className="max-w-3xl">
        <PageHeader
            title="Notifications"
            subtitle={unread > 0 ? `${unread} unread` : "Things that need you, and what's new."}
            actions={unread > 0 && <button className="btn btn-sm btn-ghost bg-white/5 hover:bg-white/10 gap-1.5" onClick={readAll}>
                <CheckCheck size={15}/> Mark all read
            </button>}
        />
        <div className="mb-4">
            <SegmentedTabs options={views.map(v => v.key === "unread" ? { ...v, count: unread } : v)}
                value={view} onChange={pickView} label="Show notifications"/>
        </div>
        {body()}
    </div>
}

/**
 * One notification: opening it goes to its link, the button beside it marks it read or unread
 * @param {object} item the notification
 * @param {function} onOpen called with the item when it's opened
 * @param {function} onToggleRead called with (item, read)
 */
function NotificationRow({ item, onOpen, onToggleRead }) {
    const { icon: Icon, label, tone } = notificationKind(item.kind)
    const isUnread = !item.read_at

    return <li className={`group flex items-stretch rounded-field ring-1 transition-colors animate-fade-up
        ${isUnread ? "bg-white/[0.06] ring-white/10" : "bg-white/[0.02] ring-white/5"}`}>
        <button type="button" onClick={() => onOpen(item)}
            className="flex-1 min-w-0 flex items-start gap-3 p-3.5 text-left rounded-field hover:bg-white/[0.03] focus-visible:outline-2 focus-visible:outline-primary">
            <span className={`grid size-9 shrink-0 place-items-center rounded-full ${tone}`} aria-hidden="true">
                <Icon size={17}/>
            </span>
            <span className="min-w-0 flex-1">
                <span className="flex items-center gap-2">
                    {isUnread && <span className="size-2 shrink-0 rounded-full bg-primary" aria-label="Unread"></span>}
                    <span className={`truncate ${isUnread ? "font-semibold" : "font-medium text-base-content/75"}`}>{item.title}</span>
                </span>
                {item.body && <span className="block text-sm text-base-content/55 mt-0.5 line-clamp-2 break-words">{item.body}</span>}
                <span className="block text-xs text-base-content/40 mt-1" title={Format.dateTime(item.created_at)}>
                    {label} · {Format.shortDateTime(item.created_at)}
                </span>
            </span>
        </button>
        <button type="button" onClick={() => onToggleRead(item, isUnread)}
            className="btn btn-ghost btn-square self-center mr-1.5 text-base-content/45 hover:text-base-content"
            aria-label={isUnread ? "Mark read" : "Mark unread"} title={isUnread ? "Mark read" : "Mark unread"}>
            {isUnread ? <CircleCheck size={17}/> : <Circle size={17}/>}
        </button>
    </li>
}

