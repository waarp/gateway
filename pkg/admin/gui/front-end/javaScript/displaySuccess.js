document.addEventListener("DOMContentLoaded", function() {
    if (window.modalOpen && window.modalOpen !== "") {
        var modal = document.getElementById(window.modalOpen);
        if (modal) {
            var collapseParent = modal.closest('.no-anim-collapse');
            if (collapseParent) {
                bootstrap.Collapse.getOrCreateInstance(collapseParent).show();
            }
            var reOpenModal = new bootstrap.Modal(modal);
            reOpenModal.show();
        }
    }

    var successPopup = document.getElementById('successPopup');
    if (successPopup) {
        setTimeout(function() {
            successPopup.classList.remove('show');
            setTimeout(function() { 
                successPopup.remove(); 
            }, 500);
        }, 3000);

        if (window.location.search.includes('success=')) {
            var url = new URL(window.location);
            url.searchParams.delete('success');
            window.history.replaceState({}, '', url);
        }
    }
});