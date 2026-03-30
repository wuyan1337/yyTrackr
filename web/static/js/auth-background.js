document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("[data-auth-slideshow]").forEach(function (container) {
        var slides = Array.from(container.querySelectorAll(".auth-stage__slide"));
        if (slides.length <= 1) {
            return;
        }

        var index = 0;
        window.setInterval(function () {
            slides[index].classList.remove("is-active");
            index = (index + 1) % slides.length;
            slides[index].classList.add("is-active");
        }, 5000);
    });
});
