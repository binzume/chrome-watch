
if (!globalThis.scriptExecuted) {
    globalThis.scriptExecuted = true;
    function install() {
        let s = document.createElement("style");
        s.innerText = "body{color:red !important}";
        document.head.appendChild(s);
    }

    if (document.readyState === "loading") {
        document.addEventListener('DOMContentLoaded', install, { once: true });
	} else {
        install();
	}
}
