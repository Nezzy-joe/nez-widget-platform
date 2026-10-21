(function () {
  "use strict";

  var script = document.currentScript;
  if (!script || !script.src) {
    return;
  }

  var scriptURL = new URL(script.src, window.location.href);
  var apiBase = scriptURL.origin;
  var scriptWidgetID = (scriptURL.searchParams.get("widget_id") || "").trim();
  var publicIDPattern = /^[0-9a-f]{36}$/;
  var styleID = "nez-widget-v1-styles";

  function element(tag, className, text) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function installStyles() {
    if (document.getElementById(styleID)) return;

    var style = document.createElement("style");
    style.id = styleID;
    style.textContent = [
      ".nez-widget-shell{box-sizing:border-box;max-width:480px;width:100%;",
      "margin:0 auto;font:16px/1.5 -apple-system,BlinkMacSystemFont,",
      "'Segoe UI',sans-serif;color:#202124}",
      ".nez-widget-shell *{box-sizing:border-box}",
      ".nez-widget-card{padding:24px;border:1px solid #e5e7eb;",
      "border-radius:18px;background:#fff;box-shadow:0 8px 30px ",
      "rgba(15,23,42,.06)}",
      ".nez-widget-title{margin:0 0 8px;font-size:24px;line-height:1.25;",
      "letter-spacing:-.035em;font-weight:650}",
      ".nez-widget-description{margin:0 0 22px;color:#626975}",
      ".nez-widget-field{margin:0 0 16px}",
      ".nez-widget-label{display:block;margin:0 0 6px;font-size:14px;",
      "font-weight:600}",
      ".nez-widget-input{display:block;width:100%;min-height:44px;",
      "padding:10px 12px;border:1px solid #d1d5db;border-radius:10px;",
      "background:#fff;color:inherit;font:inherit;outline:none}",
      ".nez-widget-input:focus{border-color:#64748b;box-shadow:0 0 0 3px ",
      "rgba(100,116,139,.17)}",
      "textarea.nez-widget-input{min-height:100px;resize:vertical}",
      ".nez-widget-checkbox{display:flex;align-items:flex-start;gap:10px}",
      ".nez-widget-checkbox input{margin-top:5px}",
      ".nez-widget-button{width:100%;min-height:46px;padding:11px 16px;",
      "border:0;border-radius:11px;background:#17191c;color:#fff;",
      "font:inherit;font-weight:600;cursor:pointer;transition:opacity .2s}",
      ".nez-widget-button:hover{opacity:.88}",
      ".nez-widget-button:disabled{opacity:.55;cursor:wait}",
      ".nez-widget-status{margin:14px 0 0;font-size:14px;color:#4b5563}",
      ".nez-widget-trap{position:absolute!important;width:1px!important;",
      "height:1px!important;padding:0!important;margin:-1px!important;",
      "overflow:hidden!important;clip:rect(0,0,0,0)!important;",
      "white-space:nowrap!important;border:0!important}"
    ].join("");

    (document.head || document.documentElement).appendChild(style);
  }

  function showMessage(root, message, isError) {
    root.replaceChildren();

    var card = element("div", "nez-widget-card");
    card.setAttribute("role", isError ? "alert" : "status");
    card.appendChild(element("p", "nez-widget-status", message));
    root.appendChild(card);
    root.setAttribute("data-nez-widget-state", isError ? "error" : "success");
  }

  function renderWidget(root, publicID, widget) {
    if (!widget || !Array.isArray(widget.fields) || widget.fields.length === 0) {
      showMessage(root, "This widget does not have any form fields configured.", true);
      return;
    }

    var card = element("section", "nez-widget-card");
    var title = element("h2", "nez-widget-title", widget.title || "Contact us");
    card.appendChild(title);

    if (widget.description) {
      card.appendChild(
        element("p", "nez-widget-description", widget.description)
      );
    }

    var form = element("form", "nez-widget-form");
    var controls = [];

    widget.fields.forEach(function (field, index) {
      if (!field || !field.name || !field.label) return;

      var fieldType = field.field_type;
      var group = element("div", "nez-widget-field");
      var id = "nez-widget-" + publicID + "-" + index;
      var control;

      if (fieldType === "textarea") {
        control = element("textarea", "nez-widget-input");
      } else {
        control = element("input", "nez-widget-input");

        var inputTypes = {
          text: "text",
          email: "email",
          tel: "tel",
          number: "number",
          url: "url",
          checkbox: "checkbox",
          select: "text"
        };

        control.type = inputTypes[fieldType] || "text";
      }

      control.id = id;
      control.name = field.name;

      if (field.required) control.required = true;

      if (fieldType !== "checkbox") {
        control.maxLength = 4096;
        control.className = "nez-widget-input";
      }

      var label = element("label", "nez-widget-label", field.label);
      label.htmlFor = id;

      if (fieldType === "checkbox") {
        group.className += " nez-widget-checkbox";
        group.appendChild(control);
        group.appendChild(label);
      } else {
        group.appendChild(label);
        group.appendChild(control);
      }

      form.appendChild(group);
      controls.push({ field: field, control: control });
    });

    if (controls.length === 0) {
      showMessage(root, "This widget does not have any supported form fields.", true);
      return;
    }

    // The existing API recognises this honeypot field.
    var trap = element("div", "nez-widget-trap");
    trap.setAttribute("aria-hidden", "true");

    var honeypot = element("input");
    honeypot.type = "text";
    honeypot.name = "company_website";
    honeypot.tabIndex = -1;
    honeypot.autocomplete = "off";
    trap.appendChild(honeypot);
    form.appendChild(trap);

    var button = element(
      "button",
      "nez-widget-button",
      widget.button_text || "Submit"
    );
    button.type = "submit";
    form.appendChild(button);

    var status = element("p", "nez-widget-status", "");
    status.setAttribute("role", "status");
    status.setAttribute("aria-live", "polite");

    form.addEventListener("submit", async function (event) {
      event.preventDefault();

      if (!form.reportValidity()) return;

      var data = {};

      for (var i = 0; i < controls.length; i++) {
        var entry = controls[i];
        var field = entry.field;
        var input = entry.control;

        if (field.field_type === "checkbox") {
          data[field.name] = input.checked;
          continue;
        }

        var value = input.value.trim();

        // The API rejects explicitly supplied empty values, even for
        // optional fields, so omit blank optional controls.
        if (value === "") {
          if (field.required) {
            input.focus();
            return;
          }
          continue;
        }

        if (field.field_type === "number") {
          var numberValue = Number(value);
          if (!Number.isFinite(numberValue)) {
            input.focus();
            return;
          }
          data[field.name] = numberValue;
        } else {
          data[field.name] = value;
        }
      }

      button.disabled = true;
      status.textContent = "Sending your submission…";

      try {
        var response = await fetch(new URL("/submissions", apiBase), {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            public_id: publicID,
            data: data,
            company_website: honeypot.value
          })
        });

        if (!response.ok) {
          if (response.status === 429) {
            throw new Error("Too many attempts. Please try again later.");
          }
          throw new Error("We couldn't send your submission. Please try again.");
        }

        showMessage(root, "Thank you! Your submission has been received.", false);
      } catch (error) {
        status.textContent = error.message ||
          "Something went wrong. Please try again.";
        button.disabled = false;
      }
    });

    card.appendChild(form);
    card.appendChild(status);
    root.replaceChildren(card);
    root.setAttribute("data-nez-widget-state", "ready");
  }

  async function initWidget(root) {
    if (root.getAttribute("data-nez-widget-initialized") === "true") return;
    root.setAttribute("data-nez-widget-initialized", "true");

    var publicID = (root.getAttribute("data-nez-widget") || "").trim();

    installStyles();
    root.classList.add("nez-widget-shell");

    if (!publicIDPattern.test(publicID)) {
      showMessage(root, "This widget ID is invalid.", true);
      return;
    }

    root.setAttribute("data-nez-widget-state", "loading");
    var loading = element("div", "nez-widget-card");
    loading.setAttribute("role", "status");
    loading.appendChild(element("p", "nez-widget-status", "Loading form…"));
    root.replaceChildren(loading);

    try {
      var response = await fetch(
        new URL("/public/widgets/" + encodeURIComponent(publicID), apiBase),
        { headers: { "Accept": "application/json" } }
      );

      if (!response.ok) {
        throw new Error("This widget could not be loaded.");
      }

      var payload = await response.json();
      renderWidget(root, publicID, payload.widget);
    } catch (error) {
      showMessage(
        root,
        error.message || "This widget could not be loaded. Please try again.",
        true
      );
    }
  }

  function boot() {
    var roots = Array.prototype.slice.call(
      document.querySelectorAll("[data-nez-widget]")
    );

    if (scriptWidgetID) {
      var root = document.createElement("div");
      root.setAttribute("data-nez-widget", scriptWidgetID);
      root.setAttribute("data-nez-widget-source", "script");

      if (
        script.parentNode &&
        script.parentNode !== document.head &&
        script.parentNode !== document.documentElement
      ) {
        script.parentNode.insertBefore(root, script.nextSibling);
      } else {
        (document.body || document.documentElement).appendChild(root);
      }

      roots.push(root);
    }

    roots.forEach(function (root) {
      initWidget(root);
    });
  }

  installStyles();
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
