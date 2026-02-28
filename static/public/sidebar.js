(function () {
  var toggle = document.getElementById("sidebar-drawer");
  if (!toggle) return;

  var STORAGE_KEY = "sidebar-expanded";
  var THEME_KEY = "theme";
  var LG = 1024;

  // Theme persistence: sync theme-controller checkbox and data-theme with localStorage
  var themeController = document.querySelector('input.theme-controller[value="bumblebee-dark"]');
  if (themeController) {
    var savedTheme = localStorage.getItem(THEME_KEY) || "bumblebee";
    themeController.checked = savedTheme === "bumblebee-dark";
    document.documentElement.setAttribute("data-theme", savedTheme);
    themeController.addEventListener("change", function () {
      var theme = themeController.checked ? "bumblebee-dark" : "bumblebee";
      localStorage.setItem(THEME_KEY, theme);
      document.documentElement.setAttribute("data-theme", theme);
      window.dispatchEvent(new CustomEvent("themechange", { detail: { theme: theme } }));
      setTimeout(function () {
        window.dispatchEvent(new Event("resize"));
      }, 50);
    });
  }

  // Persist toggle state on change (only on desktop)
  // and trigger resize so ECharts and other components recalculate
  toggle.addEventListener("change", function () {
    if (window.innerWidth >= LG) {
      localStorage.setItem(STORAGE_KEY, toggle.checked ? "true" : "false");
    }
    // Wait for the CSS transition to finish (200ms), then fire resize
    setTimeout(function () {
      window.dispatchEvent(new Event("resize"));
    }, 250);
  });

  // After page fully loads, fire resize to ensure ECharts recalculates
  // with the correct sidebar width
  window.addEventListener("load", function () {
    setTimeout(function () {
      window.dispatchEvent(new Event("resize"));
    }, 50);
  });

  // On resize: restore desktop preference when crossing to lg+
  var prev = window.innerWidth;
  window.addEventListener("resize", function () {
    var w = window.innerWidth;
    // Crossing from mobile to desktop
    if (prev < LG && w >= LG) {
      var saved = localStorage.getItem(STORAGE_KEY);
      toggle.checked = saved !== "false";
    }
    // Crossing from desktop to mobile — uncheck (hide)
    if (prev >= LG && w < LG) {
      toggle.checked = false;
    }
    prev = w;
  });
})();
