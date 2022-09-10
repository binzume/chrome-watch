// ==UserScript==
// @name         OculusBrowser theme
// @version      0.1
// @description  Set browser UI colors
// @match        chrome://panel-app-nav/*
// ==/UserScript==

let styles = {
  Default:  /* css */  '',
  Blue:  /* css */  `:root {
    --oc-toast-background: #2a2a60;
    --oc-panel-side-nav-background: #1a1a40;
    --oc-primary-text: #ffffff;
    --oc-secondary-text: #aaaaaa;
    --oc-placeholder-text: #888888;
    --oc-panel-background: #2d2d44; /* tab hover */
    --oc-card-background: #5151aa; /* popup */
  }`,
  Pink:  /* css */  `:root {
    --oc-toast-background: #f78;
    --oc-panel-side-nav-background: #c56;
    --oc-primary-text: #333;
    --oc-secondary-text: #aaa;
    --oc-placeholder-text: #666;
    --oc-panel-background: #c22; /* tab hover */
    --oc-card-background: #ffaaaa; /* popup */
    --oc-context-menu-background: #ffaaaa;
    color-scheme: light !important;
  }`,
};

let mkEl = (tag, children, attrs) => {
  let el = document.createElement(tag);
  children && el.append(...[children].flat(999));
  attrs instanceof Function ? attrs(el) : (attrs && Object.assign(el, attrs));
  return el;
};

let styleEl = document.head.appendChild(document.createElement("style"));
styleEl.textContent = styles.Blue;

function addMenuItem() {
  let menuEl = document.querySelector('.main_item_0').parentElement;
  for (let name of Object.keys(styles)) {
    menuEl.append(mkEl('button', name, {
      className: 'vertical_menu_row_button scripts-views-common-button-module__base--43dlB',
      onclick: (ev) => {
        styleEl.textContent = styles[name];
      }
    }));
  }
}

if (document.querySelector('.main_item_0')) {
  addMenuItem();
} else {
  setTimeout(addMenuItem, 500);
}
