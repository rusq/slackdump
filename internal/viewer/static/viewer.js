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
        if (event.target.closest('a, [role="tab"], [data-close-panel]')) { stopFollowing(); }
        if (event.target.closest("[data-jump-latest]")) {
            jumpToLatest();
            return;
        }
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

    var settingsKey = "slackdump.viewer.settings";
    var conversationStart = "oldest";
    var stopFollowing = function () {};

    function readSettings() {
        try {
            var value = JSON.parse(window.localStorage.getItem(settingsKey));
            if (value && value.version === 1 &&
                (value.conversationStart === "oldest" || value.conversationStart === "latest")) {
                return value.conversationStart;
            }
        } catch (_) {
            // Storage may be blocked or contain an invalid value.
        }
        return "oldest";
    }

    function conversationList() {
        return qs("#conversation #tab-panel-conversation");
    }

    function jumpToLatest() {
        stopFollowing();
        var list = conversationList();
        var content = list && qs("[data-conversation-content]", list);
        if (!content || !qs("article.message .message-header", content)) {
            return;
        }
        function alignBottom() {
            list.scrollTop = list.scrollHeight;
        }
        var observer = typeof ResizeObserver === "function" ? new ResizeObserver(alignBottom) : null;
        var scrollKeys = ["ArrowUp", "ArrowDown", "PageUp", "PageDown", "Home", "End", " "];
        function onKey(event) {
            if (scrollKeys.indexOf(event.key) !== -1) {
                stopFollowing();
            }
        }
        function onScroll() {
            if (list.scrollHeight - list.clientHeight - list.scrollTop > 2) {
                stopFollowing();
            }
        }
        stopFollowing = function () {
            if (observer) { observer.disconnect(); }
            list.removeEventListener("wheel", stopFollowing);
            list.removeEventListener("touchstart", stopFollowing);
            list.removeEventListener("pointerdown", stopFollowing);
            list.removeEventListener("scroll", onScroll);
            document.removeEventListener("keydown", onKey);
            stopFollowing = function () {};
        };
        list.addEventListener("wheel", stopFollowing, { passive: true });
        list.addEventListener("touchstart", stopFollowing, { passive: true });
        list.addEventListener("pointerdown", stopFollowing);
        list.addEventListener("scroll", onScroll);
        document.addEventListener("keydown", onKey);
        alignBottom();
        if (observer) { observer.observe(content); }
    }

    function initConversation(applyPreference) {
        var list = conversationList();
        var button = qs("[data-jump-latest]");
        if (button) {
            button.disabled = !list || !qs("article.message .message-header", list);
        }
        // Message anchors and thread deep links always win over the preference.
        if (applyPreference && !window.location.hash && conversationStart === "latest") {
            jumpToLatest();
        }
    }

    function initSettings() {
        var dialog = qs("#viewer-settings");
        var opener = qs("#open-settings");
        var select = qs("#conversation-start");
        var status = qs("#settings-status");
        if (!dialog || !opener) { return; }
        opener.addEventListener("click", function () {
            stopFollowing();
            select.value = conversationStart;
            status.hidden = true;
            dialog.showModal();
            select.focus();
        });
        qs("#cancel-settings").addEventListener("click", function () { dialog.close(); });
        dialog.addEventListener("close", function () { opener.focus(); });
        qs("#settings-form").addEventListener("submit", function (event) {
            event.preventDefault();
            conversationStart = select.value === "latest" ? "latest" : "oldest";
            try {
                window.localStorage.setItem(settingsKey, JSON.stringify({
                    version: 1, conversationStart: conversationStart
                }));
                dialog.close();
            } catch (_) {
                status.textContent = "Settings apply for this page but could not be saved.";
                status.hidden = false;
            }
        });
    }

    function init() {
        document.addEventListener("click", onDocumentClick);
        document.addEventListener("keydown", onTabKeydown);
        syncActiveChannel();
        conversationStart = readSettings();
        initSettings();
        var navigation = window.performance && performance.getEntriesByType("navigation")[0];
        initConversation(/^\/archives\/[^/]+\/?$/.test(window.location.pathname) &&
            (!navigation || navigation.type !== "back_forward"));
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

    document.body.addEventListener("htmx:beforeSwap", function (event) {
        if (event.detail && event.detail.target && event.detail.target.id === "conversation" &&
            event.detail.shouldSwap) {
            stopFollowing();
        }
    });
    document.body.addEventListener("htmx:afterSettle", function (event) {
        syncActiveChannel();
        if (event.detail && event.detail.target && event.detail.target.id === "conversation") {
            initConversation(true);
        }
    });
    document.body.addEventListener("htmx:beforeHistorySave", function () {
        var list = conversationList();
        if (list) { list.setAttribute("data-history-scroll", list.scrollTop); }
    });
    document.body.addEventListener("htmx:historyRestore", function () {
        stopFollowing();
        syncActiveChannel();
        initConversation(false);
        var list = conversationList();
        if (list && list.hasAttribute("data-history-scroll")) {
            list.scrollTop = Number(list.getAttribute("data-history-scroll")) || 0;
        }
    });
    window.addEventListener("popstate", function () { stopFollowing(); });
    window.addEventListener("hashchange", function () { stopFollowing(); });
    window.addEventListener("pagehide", function () { stopFollowing(); });

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
