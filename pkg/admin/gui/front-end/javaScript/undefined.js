document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('input.form-control-plaintext, input.form-control-plaintext[readonly], input.form-control-plaintext[disabled]').forEach(function(input) {
        if (input.value === '<non définie>' || input.value === '<undefined>') {
            input.classList.add('input-undefined');
        }
    });
});