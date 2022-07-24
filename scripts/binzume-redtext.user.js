// ==UserScript==
// @name         RedText
// @version      0.1
// @description  Make text red color at binzume.net.
// @match        https://www.binzume.net/*
// ==/UserScript==

let styleEl = document.createElement("style");
styleEl.innerText = "body{color:red !important}";
document.head.appendChild(styleEl);
