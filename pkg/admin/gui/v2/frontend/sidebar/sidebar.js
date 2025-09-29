document.addEventListener('DOMContentLoaded', () => {
    // Enable tooltips
    const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    const tooltipList = tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl)
    });

    // Set the variable for header's height
    const header = document.querySelector('header');
    if (header)
        document.documentElement.style.setProperty('--header-height', `${header.offsetHeight}px`);

    // Hide the sections that have been collapsed
    const sections = document.querySelectorAll('.sidebar-category');
    sections.forEach(section => hideSection(section));
})

function flipSectionState(button) {
    console.log(`Clicked on button ${button}`)
    const currentState = window.localStorage.getItem(button.id);
    if (currentState === 'hide') {
        window.localStorage.setItem(button.id, 'show');
    } else {
        window.localStorage.setItem(button.id, 'hide');
    }
}

function hideSection(button) {
    const state = window.localStorage.getItem(button.id);
    if (state !== 'hide') {
        return
    }

    const sectionID = button.getAttribute('data-bs-target').substring(1);
    const section = document.getElementById(sectionID);
    if (!section) {
        throw new Error(`Section with ID ${sectionID} not found`);
    }

    button.setAttribute('aria-expanded', 'false');
    section.classList.remove('show');
}
