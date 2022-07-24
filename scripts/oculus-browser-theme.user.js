// ==UserScript==
// @name         OculusBrowser theme
// @version      0.1
// @description  Set browser UI colors
// @match        chrome://panel-app-nav/*
// ==/UserScript==

let styles = {
  default:'',
  blue: `:root {
    --oc-toast-background: #2a2a60;
    --oc-panel-side-nav-background: #1a1a40;
    --oc-primary-text: #ffffff;
    --oc-secondary-text: #aaaaaa;
    --oc-placeholder-text: #888888;
    --oc-panel-background: #2d2d44; /* tab hover */
    --oc-card-background: #5151aa; /* popup */
  }`,
};

let styleEl = document.createElement("style");
styleEl.innerText = styles.blue;
document.head.appendChild(styleEl);

let buttonEl = document.createElement('button');
buttonEl.innerText = 'Default';
buttonEl.onclick = (ev) => {
  styleEl.innerText = styles.default;
};
buttonEl.className = 'vertical_menu_row_button scripts-views-common-button-module__base--43dlB';
document.querySelector('.main_item_0').parentElement.append(buttonEl);

buttonEl = document.createElement('button');
buttonEl.innerText = 'Blue';
buttonEl.onclick = (ev) => {
  styleEl.innerText = styles.blue;
};
buttonEl.className = 'vertical_menu_row_button scripts-views-common-button-module__base--43dlB';
document.querySelector('.main_item_0').parentElement.append(buttonEl);
