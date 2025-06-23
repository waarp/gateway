function eyePassword(id, btn) {
    const password = document.getElementById(id);

    password.type = password.type === "password" ? "text" : "password"

    const icon = btn.querySelector("i");
    icon.classList.toggle("fa-eye");
    icon.classList.toggle("fa-eye-slash");
}