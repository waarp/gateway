document.addEventListener("DOMContentLoaded", function() {
    if (window.modalOpen && window.modalOpen !== "") {
        var modal = document.getElementById(window.modalOpen);
        if (modal) {
            var reOpenModal = new bootstrap.Modal(modal);
            reOpenModal.show();
        }
    }

    var popup = document.getElementById('errorPopup');
    if (popup) {
        setTimeout(function() {
            popup.classList.remove('show');
            setTimeout(function() { 
                popup.remove(); 
            }, 500);
        }, 5000);
    }
});