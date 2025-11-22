import { Stats } from './stats.js';
import { Renderer } from './renderer.js';
import { WebsocketConnection } from './websocket_connection.js';
import { Packet } from './packet.js';
import { Filter } from './filter.js';

docReady(main);

function docReady(fn) {
    // see if DOM is already available
    if (document.readyState === "complete" || document.readyState === "interactive") {
        // call on next available tick
        setTimeout(fn, 1);
    } else {
        document.addEventListener("DOMContentLoaded", fn);
    }
} 

function main() {
    const toggleButton = document.getElementById('toggle');
    const conn = new WebsocketConnection(messageHandler);
    conn.connect();

    const filter = initFilter();
    Stats.instance().setFilter(filter);
    Renderer.instance().setFilter(filter);

    toggleButton.addEventListener('click', function() {
        const pauseStatus = conn.toggleConnect();
        if (pauseStatus) {
            toggleButton.innerText = 'Connect';
            toggleButton.classList.remove('connected');
        } else {
            toggleButton.innerText = 'Disconnect';
            toggleButton.classList.add('connected');
        }
    });
}

function initFilter() {
    const filter = new Filter();

    const filterInput = document.getElementById('filter');
    function b() {
        filter.setExpression(filterInput.value);
    }

    b();
    filterInput.addEventListener('input', b);

    return filter;
}

async function messageHandler(event) {
    if (!event.data.text) {
        return;
    }

    const stringData = await event.data.text();

    try {
        const packet = Packet.parse(stringData);

        Stats.instance().addPacket(packet);
        Renderer.instance().addPacket(packet);
    } catch (e) {
        console.error(e);
    }
}

