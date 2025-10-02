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

    var errorPopup = document.getElementById('errorPopup');
    if (errorPopup) {
        setTimeout(function() {
            errorPopup.classList.remove('show');
            setTimeout(function() { 
                errorPopup.remove(); 
            }, 500);
        }, 5000);
    }
});