
const maxPacketItems = 500;

main();

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
        // {"NflogPacket": {"Family":2,"Protocol":6,"PayloadLen":100,"Prefix":null,"Indev":"eth0","Outdev":"eth1","Network":{"SrcIp":"192.168.1.1","DestIp":"192.168.1.2","Protocol":6,"Transport":{"SrcPort":80,"DestPort":34}}}, "Hostname": "localhost"}
        var packet = JSON.parse(stringData);
        insertPacket(packet);
    } catch (e) { }
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

function packetToHtml(packet) {
    var html = document.createElement('div');
    html.className = 'packet';

    const hostname = packet.Hostname;
    packet = packet.NflogPacket;

    const item = html.appendChild(fieldToHtml('Hostname', hostname));
    if (item) {
        html.appendChild(item);
    }
    
    for (var key of Object.keys(packet)) {
        var value = packet[key];

        const item = html.appendChild(fieldToHtml(key, value));
        if (item) {
            html.appendChild(item);
        }
    }

    return html;
}

function fieldToHtml(key, value) {
    var div = document.createElement('div');
    div.className = 'packet-item ' + key;

    if (!value) {
        return div;
    }

    if (key === 'Network') {
        const srcIp = valueToText('SrcIp', value.SrcIp);
        const destIp = valueToText('DestIp', value.DestIp);
        const srcPort = value.Transport ? valueToText('SrcPort', value.Transport.SrcPort) : "";
        const destPort = value.Transport ? valueToText('DestPort', value.Transport.DestPort): "";

        div.innerText = `${srcIp} -> ${destIp} (${srcPort} → ${destPort})`;
    } else {
        div.innerText = valueToText(key, value);
    }

    
    return div;
}

function valueToText(key, value) {
    switch (key) {
        case 'Family':
            return familyToText(value);
        case 'Protocol':
            return protocolToText(value);
        case 'PayloadLen':
            return humanizeSize(value);
        default:
            return value;
    }

    return value;
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