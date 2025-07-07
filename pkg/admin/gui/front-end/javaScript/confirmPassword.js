function confirmPassword(password, passwordConfirm, language) {
    const pwd = document.getElementById(password);
    const confirmPwd = document.getElementById(passwordConfirm);
    if (pwd.value !== confirmPwd.value) {
        confirmPwd.classList.add('is-invalid');
        if (language === "en")
            confirmPwd.setCustomValidity('Passwords do not match');
        else if (language === "fr")
            confirmPwd.setCustomValidity('Les mots de passe ne correspondent pas');
        return false;
    } else {
        confirmPwd.classList.remove('is-invalid');
        confirmPwd.setCustomValidity('');
        return true;
    }
}