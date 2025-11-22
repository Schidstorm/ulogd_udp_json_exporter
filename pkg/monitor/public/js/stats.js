export class Stats {
    static _instance = null;

    constructor() {
        this.totalPackets = 0;
        this.packetsByLabel = {};
        this.filter = null;
    }

    /**
     * Singleton instance accessor
     * @returns {Stats} The singleton instance of Stats
     */
    static instance() {
        if (!Stats._instance) {
            Stats._instance = new Stats();
        }
        return Stats._instance;
    }

    addPacket(packet) {
        this.totalPackets++;
        this.addPacketByLabel("Hostname", packet.hostname);
        this.addPacketByLabel("Protocol", packet.protocol);
        this.addPacketByLabel("Source", packet.src);
        this.addPacketByLabel("Destination", packet.dest);
        this.render();
    }

    addPacketByLabel(type, label) {
        if (!label) {
            return
        }

        if (!this.packetsByLabel[type]) {
            this.packetsByLabel[type] = {};
        }

        if (!this.packetsByLabel[type][label]) {
            this.packetsByLabel[type][label] = 0;
        }
        this.packetsByLabel[type][label]++;
    }

    render() {
        for (let labelType in this.packetsByLabel) {
            const topN = takeTopNByValue(this.packetsByLabel[labelType], 5);
            if (Object.keys(topN).length === 0) {
                continue;
            }

            const groupSum = this.getGroupSum(labelType);
            for (let label in topN) {
                const element = this.ensureElement(labelType, label);
                element.style.setProperty('--count', topN[label]);
                element.style.setProperty('--total', groupSum);
            }

            const labelsInDom = Array.from(document
                .getElementById('stat-group-'+labelType)
                .getElementsByClassName('label-stat'))
                .map(e => e.dataset.label);
                
            const labelsInTopN = Object.keys(topN);
            const toRemove = setSubtract(new Set(labelsInDom), new Set(labelsInTopN));
            for (let label of toRemove) {
                const elem = document.getElementById('stat-'+labelType+'-'+label);
                if (elem) {
                    elem.remove();
                }
            }
                
        }
    }

    getGroupSum(type) {
        if (!this.packetsByLabel[type]) {
            return 0;
        }
        
        return Object.values(this.packetsByLabel[type]).reduce((a, b) => a + b, 0);
    }

    ensureElement(type, label) {
        const found = document.getElementById('stat-'+type+'-'+label);
        if (found) {
            return found;
        }

        const labelsStats = document.getElementById('statistics').getElementsByClassName("labels-stats")[0];

        const elemLabel = document.createElement('span');
        elemLabel.classList.add('label');
        elemLabel.innerText = label;

        const elemCount = document.createElement('div');
        elemCount.classList.add('count');

        const elem = document.createElement('div');
        elem.classList.add('label-stat');
        elem.id = 'stat-'+type+'-'+label;
        elem.dataset.label = label;
        elem.appendChild(elemLabel);
        elem.appendChild(elemCount);

        let group = document.getElementById('stat-group-'+type);
        if (!group) {
            group = document.createElement('div');
            group.id = 'stat-group-'+type;
            group.className = 'label-group';
            group.style.setProperty('--type', `"${type}"`);
        }
        group.appendChild(elem);
        
        labelsStats.appendChild(group);
        return elem;
    }

    setFilter(filter) {
        this.filter = filter;
    }
}

function setSubtract(a, b) {
    return new Set([...a].filter(x => !b.has(x)));
}

function takeTopNByValue(obj, n) {
    return Object.fromEntries(
        Object.entries(obj)
            .sort((a, b) => b[1] - a[1])
            .slice(0, n)
    );
}