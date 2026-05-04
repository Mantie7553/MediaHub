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
}