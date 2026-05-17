document.addEventListener("DOMContentLoaded", function () {
  initMobileMenu();
  initLikes();
  initSearch();
  initConfirmations();
  initFollowButtons();
  initTags();
  initScrollReveal();
  initSmoothCounters();
  initCursorGlow();
  initCardTilt();
  initPageEntrance();
  initOrbParallax();
  initMagneticButtons();
});

/* ─── Mobile Menu ────────────────────────── */
function initMobileMenu() {
  var toggle = document.querySelector("[data-menu-toggle]");
  var menu   = document.querySelector("[data-menu]");
  if (!toggle || !menu) return;

  toggle.addEventListener("click", function () {
    menu.classList.toggle("is-open");
    var spans = toggle.querySelectorAll("span");
    if (menu.classList.contains("is-open")) {
      spans[0].style.transform = "rotate(45deg) translate(5px, 5px)";
      spans[1].style.opacity   = "0";
      spans[1].style.transform = "scale(0)";
      spans[2].style.transform = "rotate(-45deg) translate(5px, -5px)";
    } else {
      spans[0].style.transform = "";
      spans[1].style.opacity   = "";
      spans[1].style.transform = "";
      spans[2].style.transform = "";
    }
  });
}

/* ─── Likes ──────────────────────────────── */
function initLikes() {
  document.querySelectorAll("[data-like-button]").forEach(function (button) {
    button.addEventListener("click", function (e) {
      var postId = button.dataset.postId;
      if (!postId || button.disabled) return;
      button.disabled = true;

      /* ripple */
      var ripple = document.createElement("span");
      ripple.className = "ripple";
      var rect = button.getBoundingClientRect();
      ripple.style.left = (e.clientX - rect.left - 10) + "px";
      ripple.style.top  = (e.clientY - rect.top  - 10) + "px";
      button.appendChild(ripple);
      setTimeout(function () { ripple.remove(); }, 700);

      var csrfMeta = document.querySelector('meta[name="csrf-token"]');
      var csrfToken = csrfMeta ? csrfMeta.content : "";
      fetch("/api/posts/" + postId + "/like", {
        method: "POST",
        headers: { Accept: "application/json", "X-CSRF-Token": csrfToken },
      })
        .then(function (res) {
          if (res.status === 401) { window.location.href = "/login"; return null; }
          if (!res.ok) throw new Error("err");
          return res.json();
        })
        .then(function (data) { if (data) updateLikeButton(button, data); })
        .catch(function () {
          button.classList.add("has-error");
          setTimeout(function () { button.classList.remove("has-error"); }, 800);
        })
        .finally(function () { button.disabled = false; });
    });
  });
}

function updateLikeButton(button, data) {
  var card  = button.closest(".post-card, .post-detail");
  var count = card ? card.querySelector("[data-like-count]") : null;
  var label = button.querySelector("[data-like-label]");
  var liked = Boolean(data.liked);

  button.classList.toggle("is-liked", liked);

  if (count && typeof data.likes_count !== "undefined") {
    animateCounter(count, parseInt(count.textContent) || 0, data.likes_count);
    count.style.color = "var(--accent)";
    setTimeout(function () { count.style.color = ""; }, 600);
  }

  if (label) label.textContent = liked ? "Liké" : "Like";

  if (liked) {
    button.style.transform = "scale(1.20)";
    setTimeout(function () { button.style.transform = ""; }, 280);
    showToast("Post liké !", "success");
  }
}

/* ─── Counter ────────────────────────────── */
function animateCounter(el, from, to) {
  var duration = 380;
  var start = performance.now();

  function tick(now) {
    var p = Math.min((now - start) / duration, 1);
    var e = 1 - Math.pow(1 - p, 4);
    el.textContent = Math.round(from + (to - from) * e);
    if (p < 1) requestAnimationFrame(tick);
  }
  requestAnimationFrame(tick);
}

/* ─── Toast ──────────────────────────────── */
function showToast(msg, type) {
  var el = document.createElement("div");
  el.className = "toast" + (type ? " toast-" + type : "");
  el.textContent = msg;
  document.body.appendChild(el);
  setTimeout(function () {
    el.style.animation = "toastOut .3s cubic-bezier(.4,0,1,1) forwards";
    setTimeout(function () { el.remove(); }, 300);
  }, 2400);
}

/* ─── Search ─────────────────────────────── */
function initSearch() {
  document.querySelectorAll("[data-search-input]").forEach(function (input) {
    var targetSelector = input.dataset.searchTarget;
    if (!targetSelector) return;
    var timer;
    input.addEventListener("input", function () {
      clearTimeout(timer);
      timer = setTimeout(function () {
        var q = normalize(input.value);
        document.querySelectorAll(targetSelector).forEach(function (item) {
          var text = normalize(item.dataset.searchText || item.textContent);
          var hide = q !== "" && text.indexOf(q) === -1;
          if (hide) {
            item.style.opacity = "0";
            item.style.transform = "scale(.96) translateY(4px)";
            item.style.transition = "opacity .15s ease, transform .15s ease";
            setTimeout(function () { item.classList.add("is-hidden"); }, 150);
          } else {
            item.classList.remove("is-hidden");
            item.style.opacity = "";
            item.style.transform = "";
            item.style.transition = "";
          }
        });
      }, 100);
    });
  });
}

/* ─── Confirmations ──────────────────────── */
function initConfirmations() {
  document.querySelectorAll("[data-confirm]").forEach(function (btn) {
    btn.addEventListener("click", function (e) {
      if (!window.confirm(btn.dataset.confirm || "Confirmer cette action ?"))
        e.preventDefault();
    });
  });
}

/* ─── Follow ─────────────────────────────── */
function initFollowButtons() {
  document.querySelectorAll(".follow-button").forEach(function (btn) {
    btn.addEventListener("click", function () {
      btn.classList.toggle("is-liked");
      var following = btn.classList.contains("is-liked");
      btn.textContent = following ? "Suivi" : "Suivre";
      btn.style.transform = "scale(.86)";
      setTimeout(function () {
        btn.style.transform = "scale(1.10)";
        setTimeout(function () { btn.style.transform = ""; }, 160);
      }, 110);
    });
  });
}

/* ─── Tags ───────────────────────────────── */
function initTags() {
  document.querySelectorAll("form").forEach(function (form) {
    var input     = form.querySelector("[data-tags-input]");
    var label     = form.querySelector("[data-selected-tags]");
    var addInput  = form.querySelector("[data-new-tag]");
    var addButton = form.querySelector("[data-add-tag]");
    var buttons   = form.querySelectorAll("[data-tag-option]");
    if (!input) return;

    function getTags() { return input.value.split(",").map(function (t) { return t.trim(); }).filter(Boolean); }

    function setTags(tags) {
      input.value = tags.join(",");
      buttons.forEach(function (b) { b.classList.toggle("is-selected", tags.indexOf(b.dataset.tagOption) !== -1); });
      if (label) label.textContent = tags.length ? "Tags : " + tags.join(", ") : "Aucun tag sélectionné";
    }

    function toggleTag(tag) { var t = getTags(); var i = t.indexOf(tag); if (i === -1) t.push(tag); else t.splice(i, 1); setTags(t); }

    buttons.forEach(function (b) {
      b.addEventListener("click", function () {
        toggleTag(b.dataset.tagOption);
        b.style.transform = "scale(.88)";
        setTimeout(function () { b.style.transform = ""; }, 160);
      });
    });

    if (addButton && addInput) {
      addButton.addEventListener("click", function () {
        var tag = addInput.value.trim();
        if (!tag) return;
        var t = getTags();
        if (t.indexOf(tag) === -1) t.push(tag);
        addInput.value = "";
        setTags(t);
      });
      addInput.addEventListener("keydown", function (e) { if (e.key === "Enter") { e.preventDefault(); addButton.click(); } });
    }
    setTags(getTags());
  });
}

/* ─── Scroll Reveal ──────────────────────── */
function initScrollReveal() {
  if (!("IntersectionObserver" in window)) return;
  var els = document.querySelectorAll(".panel, .profile-hero, .comments-section, .member-card, .empty-state");
  var obs = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        entry.target.setAttribute("data-reveal", "");
        requestAnimationFrame(function () { entry.target.classList.add("is-visible"); });
        obs.unobserve(entry.target);
      }
    });
  }, { threshold: 0.08, rootMargin: "0px 0px -28px 0px" });

  els.forEach(function (el, i) {
    el.setAttribute("data-reveal", "");
    el.style.transitionDelay = (i * 0.05) + "s";
    obs.observe(el);
  });
}

/* ─── Smooth Counters ────────────────────── */
function initSmoothCounters() {
  var els = document.querySelectorAll(".profile-stats strong");
  if (!("IntersectionObserver" in window) || !els.length) return;
  var obs = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        var el = entry.target;
        var target = parseInt(el.textContent) || 0;
        if (target > 0) { el.textContent = "0"; animateCounter(el, 0, target); }
        obs.unobserve(el);
      }
    });
  }, { threshold: .5 });
  els.forEach(function (el) { obs.observe(el); });
}

/* ─── Cursor Glow ────────────────────────── */
function initCursorGlow() {
  var glow = document.querySelector(".cursor-glow");
  if (!glow) return;
  if (window.matchMedia("(max-width: 800px)").matches) return;

  var mx = 0, my = 0, gx = 0, gy = 0;
  var size = 500;

  document.addEventListener("mousemove", function (e) {
    mx = e.clientX; my = e.clientY;
    glow.style.opacity = "1";
  });
  document.addEventListener("mouseleave", function () { glow.style.opacity = "0"; });

  /* click burst */
  document.addEventListener("mousedown", function () {
    glow.style.width  = (size * 1.3) + "px";
    glow.style.height = (size * 1.3) + "px";
    glow.style.transition = "width .15s ease, height .15s ease, opacity .4s ease";
  });
  document.addEventListener("mouseup", function () {
    glow.style.width  = size + "px";
    glow.style.height = size + "px";
  });

  function lerp(a, b, t) { return a + (b - a) * t; }

  (function tick() {
    gx = lerp(gx, mx, 0.10);
    gy = lerp(gy, my, 0.10);
    glow.style.transform = "translate(" + (gx - size / 2) + "px, " + (gy - size / 2) + "px)";
    requestAnimationFrame(tick);
  })();
}

/* ─── Card Tilt ──────────────────────────── */
function initCardTilt() {
  if (window.matchMedia("(max-width: 800px)").matches) return;
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

  document.querySelectorAll(".post-card, .member-card").forEach(function (card) {
    card.addEventListener("mousemove", function (e) {
      var rect = card.getBoundingClientRect();
      var dx   = (e.clientX - rect.left  - rect.width  / 2) / (rect.width  / 2);
      var dy   = (e.clientY - rect.top   - rect.height / 2) / (rect.height / 2);
      var rx   = -dy * 4;
      var ry   =  dx * 4;
      card.style.transform = "translateY(-4px) perspective(900px) rotateX(" + rx + "deg) rotateY(" + ry + "deg) scale(1.005)";
    });
    card.addEventListener("mouseleave", function () { card.style.transform = ""; });
  });
}

/* ─── Orb Parallax ───────────────────────── */
function initOrbParallax() {
  if (window.matchMedia("(max-width: 800px)").matches) return;
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

  var orbs = [
    { el: document.querySelector(".bg-orb-1"), fx: 0.015, fy: 0.010 },
    { el: document.querySelector(".bg-orb-2"), fx: -0.012, fy: -0.008 },
    { el: document.querySelector(".bg-orb-3"), fx: 0.020, fy: -0.015 },
    { el: document.querySelector(".bg-orb-4"), fx: -0.018, fy: 0.012 },
  ];

  var cx = window.innerWidth  / 2;
  var cy = window.innerHeight / 2;
  var tx = cx, ty = cy;
  var ox = cx, oy = cy;

  document.addEventListener("mousemove", function (e) { tx = e.clientX; ty = e.clientY; });

  (function tick() {
    ox = ox + (tx - ox) * 0.06;
    oy = oy + (ty - oy) * 0.06;
    var dx = ox - cx;
    var dy = oy - cy;

    orbs.forEach(function (o) {
      if (!o.el) return;
      var px = dx * o.fx;
      var py = dy * o.fy;
      /* combine with existing CSS animation via additional translate */
      o.el.style.marginLeft = px + "px";
      o.el.style.marginTop  = py + "px";
    });

    requestAnimationFrame(tick);
  })();
}

/* ─── Magnetic Buttons ───────────────────── */
function initMagneticButtons() {
  if (window.matchMedia("(max-width: 800px)").matches) return;
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

  document.querySelectorAll(".button.primary, .nav-login, .bottom-create").forEach(function (btn) {
    btn.addEventListener("mousemove", function (e) {
      var rect = btn.getBoundingClientRect();
      var dx = (e.clientX - rect.left - rect.width  / 2) * 0.25;
      var dy = (e.clientY - rect.top  - rect.height / 2) * 0.25;
      btn.style.transform = "translate(" + dx + "px, " + (dy - 2) + "px)";
    });
    btn.addEventListener("mouseleave", function () { btn.style.transform = ""; });
  });
}

/* ─── Page Entrance ──────────────────────── */
function initPageEntrance() {
  var heading = document.querySelector(".page-heading");
  if (!heading) return;
  heading.style.opacity   = "0";
  heading.style.transform = "translateY(14px)";
  heading.style.transition = "opacity .55s cubic-bezier(.16,1,.3,1), transform .55s cubic-bezier(.16,1,.3,1)";
  requestAnimationFrame(function () {
    requestAnimationFrame(function () {
      heading.style.opacity   = "1";
      heading.style.transform = "translateY(0)";
    });
  });
}

/* ─── Utils ──────────────────────────────── */
function normalize(v) { return v.toLowerCase().trim(); }
