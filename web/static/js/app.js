document.addEventListener("DOMContentLoaded", function () {
  initMobileMenu();
  initLikes();
  initSearch();
  initConfirmations();
  initFollowButtons();
  initTags();
  initScrollReveal();
  initSmoothCounters();
});

function initMobileMenu() {
  var toggle = document.querySelector("[data-menu-toggle]");
  var menu = document.querySelector("[data-menu]");
  if (!toggle || !menu) return;

  toggle.addEventListener("click", function () {
    menu.classList.toggle("is-open");
    var spans = toggle.querySelectorAll("span");
    if (menu.classList.contains("is-open")) {
      spans[0].style.transform = "rotate(45deg) translate(5px, 5px)";
      spans[1].style.opacity = "0";
      spans[2].style.transform = "rotate(-45deg) translate(5px, -5px)";
    } else {
      spans[0].style.transform = "";
      spans[1].style.opacity = "";
      spans[2].style.transform = "";
    }
  });
}

function initLikes() {
  var buttons = document.querySelectorAll("[data-like-button]");

  buttons.forEach(function (button) {
    button.addEventListener("click", function (e) {
      var postId = button.dataset.postId;
      if (!postId || button.disabled) return;

      button.disabled = true;

      var ripple = document.createElement("span");
      ripple.className = "ripple";
      var rect = button.getBoundingClientRect();
      ripple.style.left = (e.clientX - rect.left - 10) + "px";
      ripple.style.top = (e.clientY - rect.top - 10) + "px";
      button.appendChild(ripple);
      setTimeout(function () { ripple.remove(); }, 600);

      fetch("/api/posts/" + postId + "/like", {
        method: "POST",
        headers: { "Accept": "application/json" }
      })
        .then(function (response) {
          if (response.status === 401) {
            window.location.href = "/login";
            return null;
          }
          if (!response.ok) throw new Error("Erreur");
          return response.json();
        })
        .then(function (data) {
          if (!data) return;
          updateLikeButton(button, data);
        })
        .catch(function () {
          button.classList.add("has-error");
          setTimeout(function () { button.classList.remove("has-error"); }, 800);
        })
        .finally(function () {
          button.disabled = false;
        });
    });
  });
}

function updateLikeButton(button, data) {
  var card = button.closest(".post-card, .post-detail");
  var count = card ? card.querySelector("[data-like-count]") : null;
  var label = button.querySelector("[data-like-label]");
  var liked = Boolean(data.liked);

  button.classList.toggle("is-liked", liked);

  if (count && typeof data.likes_count !== "undefined") {
    animateCounter(count, parseInt(count.textContent) || 0, data.likes_count);
  }

  if (label) {
    label.textContent = liked ? "Lik\u00e9" : "Like";
  }

  if (liked) {
    button.style.transform = "scale(1.15)";
    setTimeout(function () { button.style.transform = ""; }, 200);
  }
}

function animateCounter(element, from, to) {
  var duration = 300;
  var start = performance.now();

  function tick(now) {
    var progress = Math.min((now - start) / duration, 1);
    var eased = 1 - Math.pow(1 - progress, 3);
    element.textContent = Math.round(from + (to - from) * eased);
    if (progress < 1) requestAnimationFrame(tick);
  }

  requestAnimationFrame(tick);
}

function initSearch() {
  var inputs = document.querySelectorAll("[data-search-input]");

  inputs.forEach(function (input) {
    var targetSelector = input.dataset.searchTarget;
    if (!targetSelector) return;

    var debounceTimer;
    input.addEventListener("input", function () {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(function () {
        var query = normalize(input.value);
        var items = document.querySelectorAll(targetSelector);

        items.forEach(function (item, index) {
          var text = normalize(item.dataset.searchText || item.textContent);
          var shouldHide = query !== "" && text.indexOf(query) === -1;

          if (shouldHide) {
            item.style.opacity = "0";
            item.style.transform = "scale(0.97)";
            setTimeout(function () { item.classList.add("is-hidden"); }, 150);
          } else {
            item.classList.remove("is-hidden");
            item.style.opacity = "";
            item.style.transform = "";
          }
        });
      }, 100);
    });
  });
}

function initConfirmations() {
  var buttons = document.querySelectorAll("[data-confirm]");

  buttons.forEach(function (button) {
    button.addEventListener("click", function (event) {
      var message = button.dataset.confirm || "Confirmer cette action ?";
      if (!window.confirm(message)) {
        event.preventDefault();
      }
    });
  });
}

function initFollowButtons() {
  var buttons = document.querySelectorAll(".follow-button");

  buttons.forEach(function (button) {
    button.addEventListener("click", function () {
      button.classList.toggle("is-liked");
      var isFollowing = button.classList.contains("is-liked");
      button.textContent = isFollowing ? "Suivi" : "Suivre";

      button.style.transform = "scale(0.92)";
      setTimeout(function () { button.style.transform = ""; }, 150);
    });
  });
}

function initTags() {
  var forms = document.querySelectorAll("form");

  forms.forEach(function (form) {
    var input = form.querySelector("[data-tags-input]");
    var label = form.querySelector("[data-selected-tags]");
    var addInput = form.querySelector("[data-new-tag]");
    var addButton = form.querySelector("[data-add-tag]");
    var buttons = form.querySelectorAll("[data-tag-option]");

    if (!input) return;

    function getTags() {
      return input.value.split(",").map(function (t) { return t.trim(); }).filter(Boolean);
    }

    function setTags(tags) {
      input.value = tags.join(",");
      buttons.forEach(function (btn) {
        btn.classList.toggle("is-selected", tags.indexOf(btn.dataset.tagOption) !== -1);
      });
      if (label) {
        label.textContent = tags.length ? "Tags : " + tags.join(", ") : "Aucun tag s\u00e9lectionn\u00e9";
      }
    }

    function toggleTag(tag) {
      var tags = getTags();
      var idx = tags.indexOf(tag);
      if (idx === -1) tags.push(tag);
      else tags.splice(idx, 1);
      setTags(tags);
    }

    buttons.forEach(function (btn) {
      btn.addEventListener("click", function () {
        toggleTag(btn.dataset.tagOption);
        btn.style.transform = "scale(0.92)";
        setTimeout(function () { btn.style.transform = ""; }, 150);
      });
    });

    if (addButton && addInput) {
      addButton.addEventListener("click", function () {
        var tag = addInput.value.trim();
        if (!tag) return;
        var tags = getTags();
        if (tags.indexOf(tag) === -1) tags.push(tag);
        addInput.value = "";
        setTags(tags);
      });

      addInput.addEventListener("keydown", function (e) {
        if (e.key === "Enter") {
          e.preventDefault();
          addButton.click();
        }
      });
    }

    setTags(getTags());
  });
}

function initScrollReveal() {
  if (!("IntersectionObserver" in window)) return;

  var revealElements = document.querySelectorAll(".panel, .profile-hero, .comments-section, .member-card, .empty-state");

  var observer = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        entry.target.style.opacity = "1";
        entry.target.style.transform = "translateY(0)";
        observer.unobserve(entry.target);
      }
    });
  }, { threshold: 0.1, rootMargin: "0px 0px -40px 0px" });

  revealElements.forEach(function (el) {
    el.style.opacity = "0";
    el.style.transform = "translateY(12px)";
    el.style.transition = "opacity 0.5s cubic-bezier(0.16, 1, 0.3, 1), transform 0.5s cubic-bezier(0.16, 1, 0.3, 1)";
    observer.observe(el);
  });
}

function initSmoothCounters() {
  var statElements = document.querySelectorAll(".profile-stats strong");

  if (!("IntersectionObserver" in window) || !statElements.length) return;

  var observer = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        var el = entry.target;
        var target = parseInt(el.textContent) || 0;
        if (target > 0) {
          el.textContent = "0";
          animateCounter(el, 0, target);
        }
        observer.unobserve(el);
      }
    });
  }, { threshold: 0.5 });

  statElements.forEach(function (el) {
    observer.observe(el);
  });
}

function normalize(value) {
  return value.toLowerCase().trim();
}
