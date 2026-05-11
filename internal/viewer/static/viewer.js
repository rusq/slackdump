(function () {
    "use strict";

    var defaultConnectionMessage = "Cannot reach the viewer server. Check that it is running and try again.";

    function qs(selector, root) {
        return (root || document).querySelector(selector);
    }

    function qsa(selector, root) {
        return Array.prototype.slice.call((root || document).querySelectorAll(selector));
    }

    function container() {
        return qs(".container");
    }

    function openSidePanel() {
        var root = container();
        if (root) {
            root.classList.add("thread-open");
        }
    }

    function closeSidePanel() {
        var root = container();
        if (root) {
            root.classList.remove("thread-open");
        }
    }

    function setActiveChannel(link) {
        qsa(".channel-sidebar .channel-list a").forEach(function (el) {
            el.classList.remove("active");
        });
        if (link) {
            link.classList.add("active");
        }
    }

    function syncActiveChannel() {
        var path = window.location.pathname;
        var match = null;

        qsa(".channel-sidebar .channel-list a").forEach(function (link) {
            var target = link.getAttribute("hx-get") || link.getAttribute("href");
            if (target && (path === target || path.indexOf(target + "/") === 0)) {
                match = link;
            }
        });
        setActiveChannel(match);
    }

    function showConnectionError(message) {
        var banner = qs("#connection-status");
        if (!banner) {
            return;
        }
        banner.textContent = message || defaultConnectionMessage;
        banner.hidden = false;
    }

    function clearConnectionError() {
        var banner = qs("#connection-status");
        if (banner) {
            banner.hidden = true;
        }
    }

    function onDocumentClick(event) {
        var close = event.target.closest("[data-close-panel]");
        if (close) {
            event.preventDefault();
            closeSidePanel();
            return;
        }

        var sidePanelLink = event.target.closest('a[hx-target="#thread"]');
        if (sidePanelLink) {
            openSidePanel();
        }

        var channelLink = event.target.closest(".channel-sidebar .channel-list a");
        if (channelLink) {
            setActiveChannel(channelLink);
        }
    }

    function onTabKeydown(event) {
        var list = event.target.closest('[role="tablist"]');
        if (!list || event.target.getAttribute("role") !== "tab") {
            return;
        }

        var tabs = qsa('[role="tab"]:not([disabled])', list);
        var idx = tabs.indexOf(document.activeElement);
        if (idx === -1 || tabs.length === 0) {
            return;
        }

        if (event.key === "ArrowRight") {
            event.preventDefault();
            tabs[(idx + 1) % tabs.length].focus();
        } else if (event.key === "ArrowLeft") {
            event.preventDefault();
            tabs[(idx - 1 + tabs.length) % tabs.length].focus();
        } else if (event.key === "Home") {
            event.preventDefault();
            tabs[0].focus();
        } else if (event.key === "End") {
            event.preventDefault();
            tabs[tabs.length - 1].focus();
        }
    }

    function init() {
        document.addEventListener("click", onDocumentClick);
        document.addEventListener("keydown", onTabKeydown);
        syncActiveChannel();
    }

    document.body.addEventListener("htmx:sendError", function () {
        showConnectionError(defaultConnectionMessage);
    });

    document.body.addEventListener("htmx:timeout", function () {
        showConnectionError("Request timed out. The viewer server may be unavailable.");
    });

    document.body.addEventListener("htmx:afterRequest", function (event) {
        if (event.detail && event.detail.successful) {
            clearConnectionError();
        }
    });

    document.body.addEventListener("htmx:beforeSwap", function (event) {
        if (!event.detail || !event.detail.xhr || !event.detail.target) {
            return;
        }
        if (event.detail.target.id === "channel-heading" && event.detail.xhr.status === 400) {
            event.detail.shouldSwap = true;
            event.detail.isError = false;
        }
    });

    document.body.addEventListener("htmx:afterSettle", syncActiveChannel);

    window.addEventListener("offline", function () {
        showConnectionError("Your browser is offline. Check your network connection.");
    });
    window.addEventListener("online", clearConnectionError);

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init);
    } else {
        init();
    }
}());
