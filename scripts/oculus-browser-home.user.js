// ==UserScript==
// @name         OculusBrowser NewTabPage
// @version      0.1
// @description  Set browser UI colors
// @match        chrome://oculus-ntp/*
// ==/UserScript==

document.getElementById('new_tab_root').style.display = 'none';

let mkEl = (tag, children, attrs) => {
    let el = document.createElement(tag);
    children && el.append(...[children].flat(999));
    attrs instanceof Function ? attrs(el) : (attrs && Object.assign(el, attrs));
    return el;
};



document.body.append(
    mkEl('h1', 'Oculus Browser Home'),
    mkEl('ul', [
        mkEl('li', mkEl('a', 'Google', { href: 'https://www.google.co.jp/' })),
        mkEl('li', mkEl('a', 'Google Map', { href: 'https://www.google.co.jp/maps/' })),
        mkEl('li', mkEl('a', 'Twitter', { href: 'https://twitter.com' })),
    ])
);
