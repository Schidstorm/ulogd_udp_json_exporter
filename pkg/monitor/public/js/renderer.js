import { Filter } from "./filter.js";

const maxPacketItems = 500;

export class Renderer {
    static _instance = null;

    constructor() {
        this.rendererdPackets = 0;
        this.filter = null;
    }

    static instance() {
        if (!Renderer._instance) {
            Renderer._instance = new Renderer();
        }
        return Renderer._instance;
    }

    addPacket(packet) {
        this.rendererdPackets++;
        var html = this._packetToHtml(packet);
        var container = document.getElementById('monitor');
        container.insertBefore(html, container.firstChild);
        
        if (this.rendererdPackets > maxPacketItems) {
            document.querySelector('#monitor .packet:last-child').remove();
            this.rendererdPackets--;
        }
    }

    _packetToHtml(packet) {
        let direction = '';
        if (packet.hook === 'prerouting' || packet.hook === 'input') {
            direction = `
                <span class="direction arrow">↢</span>
                <span class="direction left">${packet.src}</span>
            `;
        } else if (packet.hook === 'postrouting' || packet.hook === 'output') {
            direction = `
                <span class="direction arrow">↣</span>
                <span class="direction right">${packet.dest}</span>
            `;
        } else if (packet.hook === 'forward') {
            direction = `
                <span class="direction left">${packet.src}</span>
                <span class="direction arrow">⇴</span>
                <span class="direction right">${packet.dest}</span>
            `;
        } else {
            direction = `
                <span class="direction left">${packet.src}</span>
                <span class="direction arrow">↣</span>
                <span class="direction right">${packet.dest}</span>
            `;
        }

        const content = `<div class="packet ${packet.accept ? 'accept' : 'drop'}">
            <div class="labels">
                <span class="label">${packet.hostname}</span>
                <span class="label">${packet.protocol}</span>
            </div>
            ${direction}
        </div>`;
        const div = document.createElement('div');
        div.innerHTML = content;
        return div.firstChild;
    }

    /**
     * 
     * @param {Filter} filter 
     */
    setFilter(filter) {
        this.filter = filter;
    }
}