/**
 * Vanilla JS renderer for CoreUI compiler output.
 */
export class CoreUI {
  /**
   * @param {object} output - Parsed JSON output from corec.
   */
  constructor(output) {
    this.output = output || {};
    this.index = new Map();
    this.actionHandlers = new Set();
    this.host = null;
    this.shadowRoot = null;
  }

  /**
   * Renders the CoreUI tree into a shadow root attached to targetElement.
   *
   * @param {HTMLElement} targetElement - Host element for the renderer.
   * @returns {ShadowRoot}
   */
  render(targetElement) {
    if (!(targetElement instanceof HTMLElement)) {
      throw new TypeError("CoreUI.render(targetElement) expects an HTMLElement");
    }

    this.host = targetElement;
    this.shadowRoot =
      targetElement.shadowRoot || targetElement.attachShadow({ mode: "open" });

    this.index.clear();
    this.shadowRoot.replaceChildren();
    this.shadowRoot.appendChild(this.setupTheme());

    if (this.output && this.output.tree) {
      this.shadowRoot.appendChild(this.renderNode(this.output.tree));
    } else {
      this.shadowRoot.appendChild(
        this.createErrorBoundary("Missing CoreUI tree in renderer input.")
      );
    }

    return this.shadowRoot;
  }

  /**
   * Injects theme CSS variables and base styles into the shadow root.
   *
   * @returns {HTMLStyleElement}
   */
  setupTheme() {
    const style = document.createElement("style");
    const theme = this.output && this.output.theme ? this.output.theme : {};
    const declarations = [];

    Object.keys(theme)
      .sort()
      .forEach((key) => {
        const safeKey = this.sanitizeThemeKey(key);
        const resolvedValue = this.resolveThemeDefinitionValue(safeKey, theme[key]);
        if (safeKey && resolvedValue) {
          declarations.push(`--coreui-${safeKey}: ${resolvedValue};`);
        }

        const semanticValue = this.semanticThemeValue(safeKey, theme[key]);
        if (safeKey && semanticValue) {
          declarations.push(`--cui-${safeKey}: ${semanticValue};`);
        }
      });

    style.textContent = `
:host {
  display: block;
  font-family: Arial, Helvetica, sans-serif;
  ${declarations.join("\n  ")}
}
*, *::before, *::after {
  box-sizing: border-box;
}
section, div, span, button, input, img, table, caption, tbody, tr, td, th {
  font: inherit;
}
[data-coreui-error] {
  border: 1px solid #ef4444;
  background: #fef2f2;
  color: #991b1b;
  padding: 0.75rem;
  border-radius: 6px;
  font-family: monospace;
}
`;

    return style;
  }

  /**
   * Renders a single CoreUI node and its children.
   *
   * @param {object} node - CoreUI JSON node.
   * @returns {HTMLElement}
   */
  renderNode(node) {
    if (!node || typeof node !== "object") {
      return this.createErrorBoundary("Invalid node payload.");
    }

    const type = node.type;
    const attrs = this.getAttributes(node);
    let element;

    switch (type) {
      case "View":
        element = document.createElement("section");
        if (attrs.title) {
          const header = document.createElement("header");
          const title = document.createElement("h1");
          title.textContent = attrs.title;
          header.appendChild(title);
          element.appendChild(header);
        }
        break;
      case "Stack":
        element = document.createElement("div");
        break;
      case "Grid":
        element = document.createElement("div");
        break;
      case "Box":
        element = document.createElement("div");
        break;
      case "Text":
        element = document.createElement("span");
        element.textContent = this.asString(attrs.value);
        break;
      case "Input":
        element = this.createInput(node.id, attrs);
        break;
      case "Image":
        element = this.createImage(attrs);
        break;
      case "Trigger":
        element = this.createTrigger(attrs);
        break;
      case "DataTable":
        element = this.createDataTable(attrs);
        break;
      default:
        return this.createErrorBoundary(`Unknown CoreUI component: ${String(type)}`);
    }

    this.decorateElement(element, node);
    this.applyComponentStyles(element, type, attrs);
    if (attrs.hidden === true) {
      element.style.display = "none";
    }

    if (Array.isArray(node.children)) {
      node.children.forEach((child) => {
        element.appendChild(this.renderNode(child));
      });
    }

    return element;
  }

  /**
   * Returns the live DOM element for a CoreUI id, if rendered.
   *
   * @param {string} id - Component id.
   * @returns {HTMLElement|null}
   */
  getById(id) {
    return this.index.get(id) || null;
  }

  /**
   * Registers a callback for Trigger actions.
   *
   * @param {Function} callback - Receives { namespace, call, params, element }.
   * @returns {Function} Unsubscribe function.
   */
  onAction(callback) {
    if (typeof callback !== "function") {
      throw new TypeError("CoreUI.onAction(callback) expects a function");
    }

    this.actionHandlers.add(callback);
    return () => {
      this.actionHandlers.delete(callback);
    };
  }

  createInput(nodeId, attrs) {
    const wrapper = document.createElement("div");
    const inputId = this.asString(nodeId) ? `${this.asString(nodeId)}__input` : "";
    if (attrs.label) {
      const label = document.createElement("label");
      label.textContent = this.asString(attrs.label);
      if (inputId) {
        label.htmlFor = inputId;
      }
      wrapper.appendChild(label);
    }

    const input = document.createElement("input");
    input.type = this.sanitizeHtmlToken(attrs.type) || "text";
    if (inputId) {
      input.id = inputId;
    }
    if (attrs.bind) {
      input.name = this.asString(attrs.bind);
    }
    wrapper.appendChild(input);
    return wrapper;
  }

  createTrigger(attrs) {
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = this.asString(attrs.label);

    const action = attrs.action && typeof attrs.action === "object" ? attrs.action : null;
    if (action) {
      button.addEventListener("click", () => {
        const payload = {
          namespace: this.asString(action.namespace),
          call: this.asString(action.call),
          params: action.params && typeof action.params === "object" ? action.params : {},
          element: button,
        };

        this.actionHandlers.forEach((handler) => {
          handler(payload);
        });
      });
    }

    return button;
  }

  createImage(attrs) {
    const image = document.createElement("img");
    image.src = this.asString(attrs.src);
    image.alt = this.asString(attrs.alt);
    return image;
  }

  createDataTable(attrs) {
    const wrapper = document.createElement("div");
    const table = document.createElement("table");
    if (attrs.source) {
      const caption = document.createElement("caption");
      caption.textContent = this.asString(attrs.source);
      table.appendChild(caption);
    }
    table.appendChild(document.createElement("tbody"));
    wrapper.appendChild(table);
    return wrapper;
  }

  decorateElement(element, node) {
    const id = this.asString(node.id);
    if (id) {
      element.id = id;
      this.index.set(id, element);
    }

    element.dataset.coreuiType = this.asString(node.type);
  }

  applyComponentStyles(element, type, attrs) {
    switch (type) {
      case "Stack":
        element.style.display = "flex";
        element.style.flexDirection = attrs.dir === "h" ? "row" : "column";
        this.applyStyleProperty(element, "gap", attrs.gap);
        this.applyStyleProperty(element, "alignItems", attrs.align);
        break;
      case "Grid":
        element.style.display = "grid";
        this.applyGridTracks(element, "gridTemplateColumns", attrs.cols);
        this.applyGridTracks(element, "gridTemplateRows", attrs.rows);
        this.applyStyleProperty(element, "gap", attrs.gap);
        break;
      case "Box":
        this.applySemanticStyles(element, { interactive: false, elevated: true });
        this.applyVariantStyles(element, attrs.variant);
        this.applyStyleProperty(element, "padding", attrs.padding);
        this.applyStyleProperty(element, "background", attrs.background);
        if (typeof attrs.border === "number") {
          element.style.borderWidth = `${attrs.border}px`;
          element.style.borderStyle = "solid";
        }
        break;
      case "Text":
        this.applyStyleProperty(element, "fontSize", attrs.size);
        this.applyStyleProperty(element, "fontWeight", attrs.weight);
        break;
      case "Image":
        this.applyStyleProperty(element, "width", attrs.width);
        break;
      case "Input":
        this.applySemanticStyles(element.querySelector("input") || element, { interactive: true, elevated: false });
        break;
      case "Trigger":
        this.applySemanticStyles(element, { interactive: true, elevated: true });
        this.applyVariantStyles(element, attrs.variant);
        break;
      default:
        break;
    }

    if (attrs.style) {
      this.applyInlineStyleString(element, attrs.style);
    }
  }

  applyGridTracks(element, property, value) {
    if (!Array.isArray(value) || value.length === 0) {
      return;
    }

    const tracks = value
      .map((item) => this.convertUnit(item, "grid"))
      .filter(Boolean)
      .join(" ");

    if (tracks) {
      element.style[property] = tracks;
    }
  }

  applyInlineStyleString(element, styleText) {
    if (typeof styleText !== "string") {
      return;
    }

    styleText.split(";").forEach((chunk) => {
      const index = chunk.indexOf(":");
      if (index <= 0) {
        return;
      }

      const property = this.sanitizeCssProperty(chunk.slice(0, index));
      const value = chunk.slice(index + 1).trim();
      if (!property) {
        return;
      }

      const camelProperty = property.replace(/-([a-z])/g, (_, letter) =>
        letter.toUpperCase()
      );
      this.applyStyleProperty(element, camelProperty, value, property);
    });
  }

  applyStyleProperty(element, property, value, rawProperty) {
    if (value == null || value === "") {
      return;
    }

    const propertyName =
      rawProperty || property.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`);
    const cssValue = this.resolveStyleValue(propertyName, value);
    if (cssValue) {
      element.style[property] = cssValue;
    }
  }

  resolveStyleValue(property, value) {
    if (this.isColorProperty(property)) {
      const themed = this.resolveThemeToken(value);
      if (themed) {
        return themed;
      }
    }

    if (typeof value === "string") {
      const unitValue = this.convertUnit(value, "literal");
      if (unitValue) {
        return unitValue;
      }
      return this.sanitizeCssValue(value);
    }

    if (typeof value === "number") {
      return String(value);
    }

    return "";
  }

  resolveThemeToken(value) {
    const key = this.sanitizeThemeKey(value);
    if (!key) {
      return "";
    }

    if (this.output && this.output.theme && Object.prototype.hasOwnProperty.call(this.output.theme, key)) {
      return `var(--coreui-${key})`;
    }

    return "";
  }

  resolveThemeDefinitionValue(key, value) {
    const semanticValue = this.semanticThemeValue(key, value);
    if (semanticValue) {
      return semanticValue;
    }

    const tokenKey = this.sanitizeThemeKey(value);
    if (tokenKey && this.themeHas(tokenKey)) {
      return `var(--coreui-${tokenKey})`;
    }

    return this.sanitizeCssValue(value);
  }

  semanticThemeValue(key, value) {
    switch (key) {
      case "radius":
        switch (value) {
          case "none":
            return "0";
          case "sm":
            return "4px";
          case "md":
            return "8px";
          case "lg":
            return "12px";
          case "full":
            return "9999px";
          default:
            return "";
        }
      case "shadow":
        switch (value) {
          case "none":
            return "none";
          case "soft":
            return "0 10px 30px rgba(15, 23, 42, 0.12)";
          case "deep":
            return "0 18px 45px rgba(15, 23, 42, 0.22)";
          default:
            return "";
        }
      case "speed":
        switch (value) {
          case "instant":
            return "all 0s linear";
          case "smooth":
            return "all 180ms ease";
          case "lazy":
            return "all 320ms ease";
          default:
            return "";
        }
      default:
        return "";
    }
  }

  applySemanticStyles(element, options) {
    if (this.themeHas("radius")) {
      element.style.borderRadius = "var(--cui-radius)";
    }
    if (options && options.elevated && this.themeHas("shadow")) {
      element.style.boxShadow = "var(--cui-shadow)";
    }
    if (options && options.interactive && this.themeHas("speed")) {
      element.style.transition = "var(--cui-speed)";
    }
  }

  applyVariantStyles(element, variant) {
    const safeVariant = this.sanitizeThemeKey(variant);
    const primary = this.resolveThemeToken("primary");
    const text = this.resolveThemeToken("text") || "inherit";

    if (!safeVariant || !primary) {
      return;
    }

    switch (safeVariant) {
      case "primary":
        element.style.background = primary;
        element.style.borderWidth = "1px";
        element.style.borderStyle = "solid";
        element.style.borderColor = primary;
        element.style.color = text;
        break;
      case "secondary":
      case "outline":
        element.style.background = "transparent";
        element.style.borderWidth = "1px";
        element.style.borderStyle = "solid";
        element.style.borderColor = primary;
        element.style.color = primary;
        break;
      case "ghost":
        element.style.background = "transparent";
        element.style.borderWidth = "1px";
        element.style.borderStyle = "solid";
        element.style.borderColor = "transparent";
        element.style.color = primary;
        break;
      default:
        break;
    }
  }

  convertUnit(value, context) {
    if (typeof value !== "string") {
      return "";
    }

    const trimmed = value.trim();
    if (!trimmed) {
      return "";
    }

    if (trimmed === "auto" || trimmed.endsWith("px") || trimmed.endsWith("%")) {
      return this.sanitizeCssValue(trimmed);
    }

    if (trimmed.endsWith("*")) {
      const weight = trimmed.slice(0, -1).trim() || "1";
      if (!/^\d+(\.\d+)?$/.test(weight)) {
        return "";
      }
      if (context === "grid") {
        return `${weight}fr`;
      }
      return weight;
    }

    return "";
  }

  getAttributes(node) {
    if (!node || typeof node !== "object") {
      return {};
    }
    if (node.attributes && typeof node.attributes === "object") {
      return node.attributes;
    }
    if (node.props && typeof node.props === "object") {
      return node.props;
    }
    return {};
  }

  createErrorBoundary(message) {
    const error = document.createElement("div");
    error.dataset.coreuiError = "true";
    error.textContent = message;
    return error;
  }

  sanitizeCssProperty(value) {
    if (typeof value !== "string") {
      return "";
    }

    const trimmed = value.trim().toLowerCase();
    return /^[a-z-]+$/.test(trimmed) ? trimmed : "";
  }

  sanitizeCssValue(value) {
    if (typeof value !== "string") {
      return "";
    }

    const trimmed = value.trim();
    if (!trimmed) {
      return "";
    }

    if (/["';{}<>[\]\\`\n\r]/.test(trimmed)) {
      return "";
    }

    return /^[a-zA-Z0-9#%.(),\-_/+\s]+$/.test(trimmed) ? trimmed : "";
  }

  sanitizeThemeKey(value) {
    if (typeof value !== "string") {
      return "";
    }
    const trimmed = value.trim();
    return /^[a-zA-Z0-9_-]+$/.test(trimmed) ? trimmed : "";
  }

  sanitizeHtmlToken(value) {
    if (typeof value !== "string") {
      return "";
    }
    const trimmed = value.trim();
    return /^[a-zA-Z0-9:_-]+$/.test(trimmed) ? trimmed : "";
  }

  isColorProperty(property) {
    return (
      property === "background" ||
      property === "background-color" ||
      property === "color" ||
      property === "border-color" ||
      property === "fill" ||
      property === "stroke"
    );
  }

  themeHas(key) {
    return (
      this.output &&
      this.output.theme &&
      Object.prototype.hasOwnProperty.call(this.output.theme, key)
    );
  }

  asString(value) {
    return value == null ? "" : String(value);
  }
}

export default CoreUI;
