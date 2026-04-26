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
      case "Graph":
        element = this.createGraph(attrs);
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

  createGraph(attrs) {
    const figure = document.createElement("figure");
    figure.style.margin = "0";
    figure.style.display = "flex";
    figure.style.flexDirection = "column";
    figure.style.gap = "0.75rem";

    const model = this.graphModel(attrs);
    const svg = this.createSvgElement("svg");
    svg.setAttribute("viewBox", "0 0 640 240");
    svg.setAttribute("width", "100%");
    svg.setAttribute("height", this.graphHeightValue(attrs.height));
    svg.setAttribute("role", "img");
    svg.setAttribute("aria-label", `${model.type} graph`);
    svg.style.display = "block";
    svg.style.overflow = "visible";

    if (model.reference) {
      this.renderGraphPlaceholder(svg, `Awaiting ${model.reference}`);
    } else if (model.values.length === 0) {
      this.renderGraphPlaceholder(svg, "No graph data");
    } else {
      switch (model.type) {
        case "bar":
          this.renderBarGraph(svg, model);
          break;
        case "area":
          this.renderAreaGraph(svg, model);
          break;
        case "pie":
          this.renderPieGraph(svg, model);
          break;
        default:
          this.renderLineGraph(svg, model);
          break;
      }
    }

    figure.appendChild(svg);

    if (model.labels.length > 0 && model.values.length > 0) {
      const legend = document.createElement("figcaption");
      legend.style.display = "flex";
      legend.style.flexWrap = "wrap";
      legend.style.gap = "0.5rem";
      legend.style.fontSize = "0.875rem";
      legend.style.color = this.resolveThemeToken("text") || "inherit";

      model.values.forEach((value, index) => {
        const chip = document.createElement("span");
        chip.textContent = `${model.labels[index]}: ${this.formatGraphNumber(value)}`;
        chip.style.padding = "0.25rem 0.5rem";
        chip.style.border = "1px solid rgba(148, 163, 184, 0.35)";
        chip.style.borderRadius = "9999px";
        legend.appendChild(chip);
      });

      figure.appendChild(legend);
    }

    return figure;
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
      case "Graph":
        element.style.width = "100%";
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

  graphModel(attrs) {
    const values = this.graphValues(attrs.data);
    const labels = this.graphLabels(attrs.labels, values.length);
    return {
      type: this.graphType(attrs.type),
      values,
      labels,
      color: this.graphColorValue(attrs.color),
      radius: this.graphRadiusValue(),
      reference:
        typeof attrs.data === "string" && attrs.data.trim().startsWith("app:")
          ? attrs.data.trim()
          : "",
    };
  }

  graphType(value) {
    const type = this.asString(value).trim();
    switch (type) {
      case "bar":
      case "area":
      case "pie":
        return type;
      default:
        return "line";
    }
  }

  graphValues(value) {
    if (!Array.isArray(value)) {
      return [];
    }

    return value
      .map((item) => {
        if (typeof item === "number" && Number.isFinite(item)) {
          return item;
        }
        if (typeof item === "string" && item.trim() !== "") {
          const parsed = Number(item);
          if (Number.isFinite(parsed)) {
            return parsed;
          }
        }
        return null;
      })
      .filter((item) => item != null);
  }

  graphLabels(value, count) {
    if (Array.isArray(value) && value.length > 0) {
      return value.slice(0, count).map((item, index) => {
        const label = this.asString(item).trim();
        return label || `Point ${index + 1}`;
      });
    }

    return Array.from({ length: count }, (_, index) => `Point ${index + 1}`);
  }

  graphHeightValue(value) {
    const height = this.convertUnit(this.asString(value || "240px"), "literal");
    return height || "240px";
  }

  graphColorValue(value) {
    const themed = this.resolveThemeToken(value);
    if (themed) {
      return themed;
    }
    const primary = this.resolveThemeToken("primary");
    if (primary) {
      return primary;
    }
    const literal = this.sanitizeCssValue(this.asString(value));
    return literal || "#6366f1";
  }

  graphRadiusValue() {
    if (!this.output || !this.output.theme) {
      return 8;
    }

    switch (this.output.theme.radius) {
      case "none":
        return 0;
      case "sm":
        return 4;
      case "lg":
        return 12;
      case "full":
        return 9999;
      default:
        return 8;
    }
  }

  graphTransitionValue() {
    if (!this.output || !this.output.theme) {
      return "180ms ease";
    }

    switch (this.output.theme.speed) {
      case "instant":
        return "0s linear";
      case "lazy":
        return "320ms ease";
      default:
        return "180ms ease";
    }
  }

  renderLineGraph(svg, model) {
    const frame = this.graphFrame();
    this.renderGraphFrame(svg, frame);
    const points = this.graphPoints(model.values, frame);
    if (points.length === 0) {
      this.renderGraphPlaceholder(svg, "No graph data");
      return;
    }

    const path = this.createSvgElement("path");
    path.setAttribute("d", this.linePath(points));
    path.setAttribute("fill", "none");
    path.setAttribute("stroke", model.color);
    path.setAttribute("stroke-width", "4");
    path.setAttribute("stroke-linejoin", "round");
    path.setAttribute("stroke-linecap", "round");
    svg.appendChild(path);

    this.animateGraphLine(path);
    this.renderGraphDots(svg, points, model.color);
  }

  renderAreaGraph(svg, model) {
    const frame = this.graphFrame();
    this.renderGraphFrame(svg, frame);
    const points = this.graphPoints(model.values, frame);
    if (points.length === 0) {
      this.renderGraphPlaceholder(svg, "No graph data");
      return;
    }

    const area = this.createSvgElement("path");
    area.setAttribute("d", this.areaPath(points, frame));
    area.setAttribute("fill", model.color);
    area.setAttribute("fill-opacity", "0.18");
    svg.appendChild(area);

    const line = this.createSvgElement("path");
    line.setAttribute("d", this.linePath(points));
    line.setAttribute("fill", "none");
    line.setAttribute("stroke", model.color);
    line.setAttribute("stroke-width", "4");
    line.setAttribute("stroke-linejoin", "round");
    line.setAttribute("stroke-linecap", "round");
    svg.appendChild(line);

    this.animateGraphLine(line);
    this.renderGraphDots(svg, points, model.color);
  }

  renderBarGraph(svg, model) {
    const frame = this.graphFrame();
    this.renderGraphFrame(svg, frame);
    const bars = this.graphBars(model.values, frame);
    bars.forEach((bar) => {
      const rect = this.createSvgElement("rect");
      rect.setAttribute("x", String(bar.x));
      rect.setAttribute("y", String(bar.y));
      rect.setAttribute("width", String(bar.width));
      rect.setAttribute("height", String(bar.height));
      rect.setAttribute("rx", String(model.radius));
      rect.setAttribute("ry", String(model.radius));
      rect.setAttribute("fill", model.color);
      rect.setAttribute("fill-opacity", "0.9");
      svg.appendChild(rect);
    });
  }

  renderPieGraph(svg, model) {
    const values = model.values.filter((value) => value > 0);
    if (values.length === 0) {
      this.renderGraphPlaceholder(svg, "No graph data");
      return;
    }

    const colors = this.graphPalette(model.color);
    const centerX = 320;
    const centerY = 120;
    const radius = 82;
    let startAngle = -90;
    const total = values.reduce((sum, value) => sum + value, 0);

    values.forEach((value, index) => {
      const sliceAngle = (value / total) * 360;
      const endAngle = startAngle + sliceAngle;
      const path = this.createSvgElement("path");
      path.setAttribute(
        "d",
        this.describePieSlice(centerX, centerY, radius, startAngle, endAngle)
      );
      path.setAttribute("fill", colors[index % colors.length]);
      path.setAttribute("stroke", "rgba(255, 255, 255, 0.9)");
      path.setAttribute("stroke-width", "2");
      svg.appendChild(path);
      startAngle = endAngle;
    });
  }

  renderGraphPlaceholder(svg, message) {
    const rect = this.createSvgElement("rect");
    rect.setAttribute("x", "16");
    rect.setAttribute("y", "16");
    rect.setAttribute("width", "608");
    rect.setAttribute("height", "208");
    rect.setAttribute("rx", "16");
    rect.setAttribute("fill", "rgba(148, 163, 184, 0.12)");
    svg.appendChild(rect);

    const text = this.createSvgElement("text");
    text.setAttribute("x", "320");
    text.setAttribute("y", "124");
    text.setAttribute("text-anchor", "middle");
    text.setAttribute("font-size", "16");
    text.setAttribute("fill", this.resolveThemeToken("text") || "#475569");
    text.textContent = message;
    svg.appendChild(text);
  }

  renderGraphFrame(svg, frame) {
    const axis = this.createSvgElement("path");
    axis.setAttribute(
      "d",
      `M ${frame.left} ${frame.top} L ${frame.left} ${frame.bottom} L ${frame.right} ${frame.bottom}`
    );
    axis.setAttribute("fill", "none");
    axis.setAttribute("stroke", "rgba(148, 163, 184, 0.5)");
    axis.setAttribute("stroke-width", "2");
    svg.appendChild(axis);
  }

  renderGraphDots(svg, points, color) {
    points.forEach((point) => {
      const dot = this.createSvgElement("circle");
      dot.setAttribute("cx", String(point.x));
      dot.setAttribute("cy", String(point.y));
      dot.setAttribute("r", "4");
      dot.setAttribute("fill", color);
      svg.appendChild(dot);
    });
  }

  graphFrame() {
    return { left: 40, top: 20, right: 620, bottom: 200 };
  }

  graphPoints(values, frame) {
    if (!Array.isArray(values) || values.length === 0) {
      return [];
    }

    const min = Math.min(...values, 0);
    const max = Math.max(...values, 1);
    const span = max - min || 1;
    const width = frame.right - frame.left;
    const height = frame.bottom - frame.top;

    return values.map((value, index) => {
      const x =
        frame.left + (values.length === 1 ? width / 2 : (width * index) / (values.length - 1));
      const y = frame.bottom - ((value - min) / span) * height;
      return { x, y };
    });
  }

  graphBars(values, frame) {
    if (!Array.isArray(values) || values.length === 0) {
      return [];
    }

    const max = Math.max(...values, 1);
    const width = frame.right - frame.left;
    const height = frame.bottom - frame.top;
    const gap = 12;
    const barWidth = Math.max((width - gap * (values.length - 1)) / values.length, 12);

    return values.map((value, index) => {
      const barHeight = max === 0 ? 0 : (value / max) * height;
      return {
        x: frame.left + index * (barWidth + gap),
        y: frame.bottom - barHeight,
        width: barWidth,
        height: barHeight,
      };
    });
  }

  linePath(points) {
    if (points.length === 0) {
      return "";
    }
    return points
      .map((point, index) => `${index === 0 ? "M" : "L"} ${point.x} ${point.y}`)
      .join(" ");
  }

  areaPath(points, frame) {
    if (points.length === 0) {
      return "";
    }
    return `${this.linePath(points)} L ${points[points.length - 1].x} ${frame.bottom} L ${points[0].x} ${frame.bottom} Z`;
  }

  animateGraphLine(path) {
    const length = typeof path.getTotalLength === "function" ? path.getTotalLength() : 0;
    if (!length) {
      return;
    }

    path.style.strokeDasharray = String(length);
    path.style.strokeDashoffset = String(length);
    path.style.transition = `stroke-dashoffset ${this.graphTransitionValue()}`;
    const schedule =
      typeof window !== "undefined" && typeof window.requestAnimationFrame === "function"
        ? window.requestAnimationFrame.bind(window)
        : (callback) => setTimeout(callback, 0);
    schedule(() => {
      path.style.strokeDashoffset = "0";
    });
  }

  graphPalette(baseColor) {
    return [
      baseColor,
      "rgba(99, 102, 241, 0.82)",
      "rgba(14, 165, 233, 0.78)",
      "rgba(16, 185, 129, 0.78)",
      "rgba(245, 158, 11, 0.8)",
    ];
  }

  describePieSlice(centerX, centerY, radius, startAngle, endAngle) {
    const start = this.polarToCartesian(centerX, centerY, radius, endAngle);
    const end = this.polarToCartesian(centerX, centerY, radius, startAngle);
    const largeArcFlag = endAngle - startAngle <= 180 ? "0" : "1";
    return [
      "M",
      centerX,
      centerY,
      "L",
      start.x,
      start.y,
      "A",
      radius,
      radius,
      0,
      largeArcFlag,
      0,
      end.x,
      end.y,
      "Z",
    ].join(" ");
  }

  polarToCartesian(centerX, centerY, radius, angleInDegrees) {
    const angleInRadians = ((angleInDegrees - 90) * Math.PI) / 180.0;
    return {
      x: centerX + radius * Math.cos(angleInRadians),
      y: centerY + radius * Math.sin(angleInRadians),
    };
  }

  createSvgElement(name) {
    return document.createElementNS("http://www.w3.org/2000/svg", name);
  }

  formatGraphNumber(value) {
    return Number.isInteger(value) ? String(value) : value.toFixed(2).replace(/\.?0+$/, "");
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
