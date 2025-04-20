/**
 * DOM Builder 工具库
 * 功能：简化动态 DOM 操作 + DocumentFragment 性能优化
 */

/**
 * 转义 HTML 特殊字符（防 XSS）
 */
const escapeHtml = (str) => {
    if (typeof str !== 'string') return str;
    return str.replace(/[&<>"']/g,
        tag => ({
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#39;'
        }[tag])
    );
};

/**
 * 创建 DOM 元素
 * @param {string} tag 标签名
 * @param {Object} [options] 配置项
 * @param {string|Array} [options.classes] 类名
 * @param {Object} [options.attrs] 属性
 * @param {Object} [options.styles] 样式
 * @param {string|Node} [options.text] 文本或节点
 * @param {Array} [options.children] 子元素数组
 * @param {Object} [options.on] 事件监听
 * @returns {HTMLElement}
 */
const createElement = (tag, options = {}) => {
    const el = document.createElement(tag);

    // ID设置
    if (options.id) {
        el.id = options.id;
    }

    // 类名处理（支持字符串或数组）
    if (options.classes) {
        const classes = Array.isArray(options.classes)
            ? options.classes
            : options.classes.split(' ');
        el.classList.add(...classes.filter(c => c));
    }

    // 属性设置
    if (options.attrs) {
        for (const [key, value] of Object.entries(options.attrs)) {
            el.setAttribute(key, escapeHtml(value));
        }
    }

    // 样式设置
    if (options.styles) {
        Object.assign(el.style, options.styles);
    }

    // 文本/子节点
    if (options.text !== undefined) {
        el.append(
            typeof options.text === 'string'
                ? document.createTextNode(escapeHtml(options.text))
                : options.text
        );
    }

    // 子元素递归创建
    if (options.children) {
        el.append(...options.children.map(child =>
            child instanceof Node ? child : createElement(...child)
        ));
    }

    // 事件监听
    if (options.on) {
        for (const [event, handler] of Object.entries(options.on)) {
            el.addEventListener(event, handler);
        }
    }

    return el;
};

/**
 * 批量创建元素到 DocumentFragment
 * @param {Array} elements 元素配置数组
 * @returns {DocumentFragment}
 */
const createFragment = (elements) => {
    const fragment = document.createDocumentFragment();
    elements.forEach(item => {
        const [tag, options] = Array.isArray(item) ? item : [item.tag, item];
        fragment.appendChild(createElement(tag, options));
    });
    return fragment;
};

// ==================== 链式调用 Builder ====================
class ElementBuilder {
    constructor(tag) {
        this.element = document.createElement(tag);
    }

    id(id) {
        this.element.id = id;
        return this;
    }

    class(...classes) {
        this.element.classList.add(...classes.flatMap(c =>
            typeof c === 'string' ? c.split(' ') : c
        ).filter(Boolean));
        return this;
    }

    attr(key, value) {
        this.element.setAttribute(key, escapeHtml(value));
        return this;
    }

    style(styles) {
        Object.assign(this.element.style, styles);
        return this;
    }

    text(content) {
        this.element.textContent = escapeHtml(content);
        return this;
    }

    html(content) {
        this.element.innerHTML = content; // 注意：此方法不安全，需确保 content 可信
        return this;
    }

    on(event, handler, options) {
        this.element.addEventListener(event, handler, options);
        return this;
    }

    append(...children) {
        children.forEach(child => {
            this.element.appendChild(
                child instanceof Node ? child : createElement(...child)
            );
        });
        return this;
    }

    build() {
        return this.element;
    }
}

// ==================== 导出 ====================
export default {
    createElement,
    createFragment,
    escapeHtml,
    builder: (tag) => new ElementBuilder(tag),

    // 快捷方法
    div: (options) => createElement('div', options),
    button: (options) => createElement('button', options),
    span: (options) => createElement('span', options),
    // 可继续扩展其他标签...
};