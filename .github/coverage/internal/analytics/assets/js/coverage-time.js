/**
 * Dynamic Timestamp Management for Coverage Reports
 * 
 * Converts static timestamps to human-readable relative time displays
 * with hover tooltips showing full formatted timestamps.
 * Updates automatically every 60 seconds to stay current.
 */

(function() {
    'use strict';

    // Configuration
    const UPDATE_INTERVAL = 60000; // 60 seconds
    const TIMESTAMP_SELECTOR = '.dynamic-timestamp';
    
    /**
     * Formats a date into a human-readable full timestamp
     * @param {Date} date - The date to format
     * @returns {string} Formatted timestamp like "January 2, 2006 at 3:04 PM UTC"
     */
    function formatFullTimestamp(date) {
        const options = {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: 'numeric',
            minute: '2-digit',
            timeZone: 'UTC',
            timeZoneName: 'short',
            hour12: true
        };
        
        return date.toLocaleDateString('en-US', options).replace(',', ' at');
    }
    
    /**
     * Calculates relative time from now
     * @param {Date} date - The date to calculate relative time for
     * @returns {string} Relative time like "2 hours ago" or "5 minutes ago"
     */
    function getRelativeTime(date) {
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        
        // Handle future dates (shouldn't happen but good to be safe)
        if (diffMs < 0) {
            return 'just now';
        }
        
        const seconds = Math.floor(diffMs / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        const weeks = Math.floor(days / 7);
        const months = Math.floor(days / 30.44); // Average month length
        const years = Math.floor(days / 365.25); // Account for leap years
        
        // Return appropriate relative time
        if (years > 0) {
            return years === 1 ? '1 year ago' : `${years} years ago`;
        } else if (months > 0) {
            return months === 1 ? '1 month ago' : `${months} months ago`;
        } else if (weeks > 0) {
            return weeks === 1 ? '1 week ago' : `${weeks} weeks ago`;
        } else if (days > 0) {
            return days === 1 ? '1 day ago' : `${days} days ago`;
        } else if (hours > 0) {
            return hours === 1 ? '1 hour ago' : `${hours} hours ago`;
        } else if (minutes > 0) {
            return minutes === 1 ? '1 minute ago' : `${minutes} minutes ago`;
        } else if (seconds > 10) {
            return `${seconds} seconds ago`;
        } else {
            return 'just now';
        }
    }
    
    /**
     * Updates a single timestamp element
     * @param {HTMLElement} element - The element containing the timestamp
     */
    function updateTimestampElement(element) {
        const timestampStr = element.getAttribute('data-timestamp');
        if (!timestampStr) {
            console.warn('Dynamic timestamp element missing data-timestamp attribute:', element);
            return;
        }
        
        try {
            const date = new Date(timestampStr);
            
            // Validate the date
            if (isNaN(date.getTime())) {
                console.warn('Invalid timestamp in data-timestamp attribute:', timestampStr);
                return;
            }
            
            // Update the relative time display
            const relativeTime = getRelativeTime(date);
            const currentText = element.textContent;
            
            // Only update if the text has changed to avoid unnecessary DOM updates
            if (currentText !== `Generated ${relativeTime}`) {
                element.textContent = `Generated ${relativeTime}`;
            }
            
            // Update or set the tooltip with full timestamp
            const fullTimestamp = formatFullTimestamp(date);
            if (element.title !== fullTimestamp) {
                element.title = fullTimestamp;
            }
            
        } catch (error) {
            console.error('Error updating timestamp element:', error, element);
        }
    }
    
    /**
     * Updates all dynamic timestamp elements on the page
     */
    function updateAllTimestamps() {
        const elements = document.querySelectorAll(TIMESTAMP_SELECTOR);
        elements.forEach(updateTimestampElement);
    }
    
    /**
     * Initialize the dynamic timestamp system
     */
    function initializeTimestamps() {
        // Update timestamps immediately
        updateAllTimestamps();
        
        // Set up periodic updates
        setInterval(updateAllTimestamps, UPDATE_INTERVAL);
        
        console.log(`Dynamic timestamps initialized. Updates every ${UPDATE_INTERVAL / 1000} seconds.`);
    }
    
    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initializeTimestamps);
    } else {
        // DOM is already ready
        initializeTimestamps();
    }
    
    // Expose functions for debugging if needed
    if (typeof window !== 'undefined') {
        window.coverageTime = {
            updateAll: updateAllTimestamps,
            getRelativeTime: getRelativeTime,
            formatFullTimestamp: formatFullTimestamp
        };
    }
    
})();