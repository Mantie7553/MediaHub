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

    static statusLabel(status, type) {
        if (!status) return "Add to list";
        
        const labels = {
            current: { anime: "Watching", manga: "Reading", light_novel: "Reading", movie: "Watching" },
            planned: { anime: "Plan to Watch", manga: "Plan to Read", light_novel: "Plan to Read", movie: "Plan to Watch" },
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