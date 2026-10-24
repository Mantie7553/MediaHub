export default class Format {
    /**
     * Format Dates as -- YYYY Month, dd at HH:MM
     * @param {any} date the date to format
     * @returns a date in the proper display format
     */
    static dateTime(date) {
        return new Intl.DateTimeFormat('en-US', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: 'numeric',
            hour12: true,
            minute: '2-digit'
        }).format(new Date(date));
    }

    /**
     * Format Dates compactly as -- Oct 18, 9:38 PM, with the year only when it isn't this year
     * @param {any} date the date to format
     * @returns a date in the short display format
     */
    static shortDateTime(date) {
        const d = new Date(date);
        const thisYear = d.getFullYear() === new Date().getFullYear();
        return new Intl.DateTimeFormat('en-US', {
            month: 'short',
            day: 'numeric',
            ...(thisYear ? {} : { year: 'numeric' }),
            hour: 'numeric',
            minute: '2-digit',
        }).format(d);
    }

    /**
     * Format Dates as -- YYYY Month, dd
     * @param {any} date the date to format
     * @returns a date in the proper display format
     */
    static date(date) {
        return new Intl.DateTimeFormat('en-US', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
        }).format(new Date(date));
    }

    /**
     * Format Dates as -- YYYY
     * @param {any} date the date to format
     * @returns a date in the proper display format
     */
    static year(date) {
        return new Intl.DateTimeFormat('en-US', {
            year: 'numeric'
        }).format(new Date(date));
    }

    /**
     * Format string to display well
     * @param {any} string the string we are formatting
     * @returns a string in the proper display format
     */
    static cleanString(string) {
        return string.replaceAll("_", " ").replace(/^\w/, c => c.toUpperCase());
    }

    /**
     * Format a playback position as -- H:MM:SS, or M:SS under an hour
     * @param {number} secs seconds into the video
     * @returns the position in the proper display format
     */
    static duration(secs) {
        const total = Math.max(0, Math.floor(secs));
        const h = Math.floor(total / 3600);
        const m = Math.floor((total % 3600) / 60);
        const s = String(total % 60).padStart(2, "0");
        return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${s}` : `${m}:${s}`;
    }

    /**
     * Format a size in bytes as -- 1.4 TB, 820 MB, 12 KB
     * @param {number} n bytes
     * @returns the size with its unit, one decimal under 10 of a unit; "—" when unknown
     */
    static bytes(n) {
        if (n == null || !Number.isFinite(n)) return "—";
        const units = ["B", "KB", "MB", "GB", "TB", "PB"];
        let value = Math.max(0, n);
        let unit = 0;
        while (value >= 1024 && unit < units.length - 1) {
            value /= 1024;
            unit++;
        }
        const digits = unit > 0 && value < 10 ? 1 : 0;
        return `${value.toFixed(digits)} ${units[unit]}`;
    }

    /**
     * Format how long ago a date was as -- just now, 5 min ago, 3 h ago, 2 days ago, then the date
     * @param {any} date the date
     * @param {Date} now the time to measure from
     * @returns the age in the proper display format; "Never" when there is no date
     */
    static relativeTime(date, now = new Date()) {
        if (!date) return "Never";
        const secs = Math.max(0, (now - new Date(date)) / 1000);
        if (secs < 60) return "just now";
        if (secs < 3600) return `${Math.floor(secs / 60)} min ago`;
        if (secs < 86400) return `${Math.floor(secs / 3600)} h ago`;
        const days = Math.floor(secs / 86400);
        if (days < 30) return days === 1 ? "yesterday" : `${days} days ago`;
        return Format.date(date);
    }

    static statusLabel(status, type) {
        if (!status) return "Add to list";
        
        const labels = {
            current: { anime: "Watching", manga: "Reading", light_novel: "Reading", movie: "Watching" },
            planned: { anime: "Plan to Watch", manga: "Plan to Read", light_novel: "Plan to Read", movie: "Plan to Watch", music_album: "Plan to Listen" },
            completed: { music_album: "Listened", default: "Completed" },
            dropped: "Dropped",
            wishlist: "Wishlist",
        }

        const entry = labels[status];
        if (!entry) return Format.cleanString(status);
        if (typeof entry === "string") return entry;
        return entry[type] ?? Format.cleanString(status);
    }
}