// ==UserScript==
// @name         RedText
// @version      0.1
// @description  Make text red color at binzume.net.
// @match        https://www.binzume.net/*
// ==/UserScript==

function install() {
    let s = document.createElement("style");
    s.innerText = "body{color:red !important}";
    document.head.appendChild(s);
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', install, { once: true });
} else {
    install();
}
