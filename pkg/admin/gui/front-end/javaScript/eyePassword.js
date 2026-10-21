function eyePassword(id, btn) {
    const password = document.getElementById(id);

    password.type = password.type === "password" ? "text" : "password"

    const icon = btn.querySelector("i");
    icon.classList.toggle("bi-eye");
    icon.classList.toggle("bi-eye-slash");
}