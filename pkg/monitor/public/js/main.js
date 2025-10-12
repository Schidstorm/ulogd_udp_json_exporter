
const maxPacketItems = 500;

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


async function messageHandler(event) {
    if (!event.data.text) {
        return;
    }

    const stringData = await event.data.text();

    try {
        /* 
            {"metadata":{"hostname":"wlan","capture_length":60,"length":60,"prefix":"accept"},"layers":[
                {"Layer":{"Ethernet":{"ethertype":2048}}},
                {"type":1,"Layer":{"Ipv4":{"src_ip":"192.168.1.9","dest_ip":"192.168.1.155","protocol":17,"ttl":64}}},
                {"type":4,"Layer":{"Udp":{"src_port":37177,"dest_port":53,"length":40,"checksum":33838}}}]}
        */
        let packet = JSON.parse(stringData);
        packet = compactPacket(packet);
        insertPacket(packet);
    } catch (e) { }
}

function compactPacket(packet) {
    let result = {};
    result = { ...packet.metadata };
    for (let layer of packet.layers) {
        layer = layer.Layer;
        const layerType = Object.keys(layer)[0];
        result = { ...result, ...layer[layerType] };
    }
    return result;
}

class Packet {
    constructor(data) {
        console.log(data);
        this.hostname = data.hostname || '';
        this.src = `${data.src_ip || ''}${data.src_port ? ':'+data.src_port : ''}`;
        this.dest = `${data.dest_ip || ''}${data.dest_port ? ':'+data.dest_port : ''}`;

        let transportProtocol = data.protocol || data.next_header || '';
        this.protocol = humanizeService(data.dest_port, transportProtocol) || transportProtocol || '';
        this.length = humanizeSize(data.length) || '';
        this.hook = data.hook || '';
        this.accept = data.prefix === 'accept';
    }
}

function humanizeService(port, protocol) {
  if  (!port || !protocol) {
    return undefined;
  }

  if (!serviceMap) {
    serviceMap = {};
    for (let service of services) {
      serviceMap[`${service.port}/${service.protocol}`] = service.name;
    }
  }

  return serviceMap[`${port}/${protocol.toLowerCase()}`] || `${port}/${protocol}`;
}

function insertPacket(packet) {
    var html = packetToHtml(packet);
    var container = document.getElementById('monitor');
    container.insertBefore(html, container.firstChild);

    
    const packetItems = container.getElementsByClassName('packet')
    if (packetItems.length > maxPacketItems) {
        container.removeChild(packetItems[packetItems.length - 1]);
    }
}

function packetToHtml(rawPacket) {
    const packet = new Packet(rawPacket);

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

function familyToText(family) {
    switch (family) {
        case 1:
            return 'IPv4';
        case 2:
            return 'IPv6';
        default:
            return family;
    }
}

function protocolToText(protocol) {
    switch (protocol) {
        case 1:
            return 'ICMP';
        case 2:
            return 'IGMP';
        case 6:
            return 'TCP';
        case 17:
            return 'UDP';
        default:
            return protocol;
    }
}

function humanizeSize(size) {
    var i = Math.floor(Math.log(size) / Math.log(1000));
    return (size / Math.pow(1000, i)).toFixed(2) + ' ' + ['B', 'KB', 'MB', 'GB', 'TB'][i];
}