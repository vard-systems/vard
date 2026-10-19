// Applies the reader's saved theme before the first paint, so a dark page
// never flashes light. A file rather than an inline script, so the Content
// Security Policy needs no exception for it. The cookie is this app's own
// (each app is on its own host); without one, prefers-color-scheme decides.
(function () {
	var saved = document.cookie.match(/(?:^|; )[\w-]+-theme=(dark|light)(?:;|$)/);
	if (saved) document.documentElement.setAttribute('data-theme', saved[1]);
})();
