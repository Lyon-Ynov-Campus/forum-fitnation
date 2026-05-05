document.addEventListener("DOMContentLoaded", function () {
  initMobileMenu();
  initLikes();
  initSearch();
  initConfirmations();
  initFollowButtons();
  initTags();
});

function initMobileMenu() {
  var toggle = document.querySelector("[data-menu-toggle]");
  var menu = document.querySelector("[data-menu]");

  if (!toggle || !menu) {
    return;
  }

  toggle.addEventListener("click", function () {
    menu.classList.toggle("is-open");
  });
}

function initLikes() {
  var buttons = document.querySelectorAll("[data-like-button]");

  buttons.forEach(function (button) {
    button.addEventListener("click", function () {
      var postId = button.dataset.postId;

      if (!postId || button.disabled) {
        return;
      }

      button.disabled = true;

      fetch("/api/posts/" + postId + "/like", {
        method: "POST",
        headers: {
          "Accept": "application/json"
        }
      })
        .then(function (response) {
          if (response.status === 401) {
            window.location.href = "/login";
            return null;
          }
          if (!response.ok) {
            throw new Error("Erreur pendant le like");
          }
          return response.json();
        })
        .then(function (data) {
          if (!data) {
            return;
          }
          updateLikeButton(button, data);
        })
        .catch(function () {
          button.classList.add("has-error");
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
    count.textContent = data.likes_count;
  }

  if (label) {
    label.textContent = liked ? "Liké" : "Like";
  }
}

function initSearch() {
  var inputs = document.querySelectorAll("[data-search-input]");

  inputs.forEach(function (input) {
    var targetSelector = input.dataset.searchTarget;

    if (!targetSelector) {
      return;
    }

    input.addEventListener("input", function () {
      var query = normalize(input.value);
      var items = document.querySelectorAll(targetSelector);

      items.forEach(function (item) {
        var text = normalize(item.dataset.searchText || item.textContent);
        item.classList.toggle("is-hidden", query !== "" && text.indexOf(query) === -1);
      });
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
      button.textContent = button.classList.contains("is-liked") ? "Suivi" : "Suivre";
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

    if (!input) {
      return;
    }

    function getTags() {
      return input.value
        .split(",")
        .map(function (tag) {
          return tag.trim();
        })
        .filter(function (tag) {
          return tag !== "";
        });
    }

    function setTags(tags) {
      input.value = tags.join(",");

      buttons.forEach(function (button) {
        button.classList.toggle("is-selected", tags.indexOf(button.dataset.tagOption) !== -1);
      });

      if (label) {
        label.textContent = tags.length ? "Tags : " + tags.join(", ") : "Aucun tag sélectionné";
      }
    }

    function toggleTag(tag) {
      var tags = getTags();
      var index = tags.indexOf(tag);

      if (index === -1) {
        tags.push(tag);
      } else {
        tags.splice(index, 1);
      }

      setTags(tags);
    }

    buttons.forEach(function (button) {
      button.addEventListener("click", function () {
        toggleTag(button.dataset.tagOption);
      });
    });

    if (addButton && addInput) {
      addButton.addEventListener("click", function () {
        var tag = addInput.value.trim();

        if (tag === "") {
          return;
        }

        var tags = getTags();
        if (tags.indexOf(tag) === -1) {
          tags.push(tag);
        }

        addInput.value = "";
        setTags(tags);
      });
    }

    setTags(getTags());
  });
}

function normalize(value) {
  return value.toLowerCase().trim();
}
