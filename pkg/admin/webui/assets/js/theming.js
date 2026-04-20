(function() {
    const saved = localStorage.getItem('theme');
    if (saved === 'dark' || (!saved && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        document.documentElement.classList.add('dark');
    }
})();

function toggleTheme() {
    const isDark = document.documentElement.classList.toggle('dark');
    const osPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    if (isDark === osPrefersDark) {
        localStorage.removeItem('theme');
    } else {
        localStorage.setItem('theme', isDark ? 'dark' : 'light');
    }
}