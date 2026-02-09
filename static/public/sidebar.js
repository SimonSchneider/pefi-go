(function () {
  var toggle = document.getElementById("sidebar-drawer");
  if (!toggle) return;

  var STORAGE_KEY = "sidebar-expanded";
  var LG = 1024;

  // On desktop (lg+), restore saved preference
  if (window.innerWidth >= LG) {
    var saved = localStorage.getItem(STORAGE_KEY);
    // Default to checked (expanded) if no preference saved
    toggle.checked = saved !== "false";
  }

  // Persist toggle state on change (only on desktop)
  toggle.addEventListener("change", function () {
    if (window.innerWidth >= LG) {
      localStorage.setItem(STORAGE_KEY, toggle.checked ? "true" : "false");
    }
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
