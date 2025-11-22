import { services } from './services.js';


let serviceMap = null;

export class Packet {
  constructor(rawData) {
    const data = this._compactPacket(rawData);

    this._data = data;
    this.hostname = data.hostname || '';
    this.src = `${data.src_ip || ''}${data.src_port ? ':' + data.src_port : ''}`;
    this.dest = `${data.dest_ip || ''}${data.dest_port ? ':' + data.dest_port : ''}`;

    let transportProtocol = data.protocol || data.next_header || '';
    this.protocol = humanizeService(data.dest_port, transportProtocol) || transportProtocol || '';
    this.length = humanizeSize(data.length) || '';
    this.hook = data.hook || '';
    this.accept = data.prefix === 'accept';
  }

  static parse(stringData) {
    return new Packet(JSON.parse(stringData));
  }

  /***
   * Compacts a packet structure by merging all layers into a single flat object.
   * Assumes that each layer has a unique key.
   * @param {Object} packet - The original packet object with metadata and layers.
   */
  _compactPacket(packet) {
    let result = {};
    result = { ...packet.metadata };
    for (let layer of packet.layers) {
      layer = layer.Layer;
      const layerType = Object.keys(layer)[0];
      result = { ...result, ...layer[layerType] };
    }
    return result;
  }

  matchFilter(rpnFilter) {
    if (rpnFilter.length === 0) {
      return true;
    }

    const stack = [];
    for (const token of rpnFilter) {
      if (token.type === 'arit-operator') {
        const right = stack.pop();
        const left = stack.pop();
        let res = false;
        switch (token.value) {
          case '==':
            res = left == right;
            break;
          case '=':
            res = left == right;
            break;
          case '>':
            res = left > right;
            break;
          case '<':
            res = left < right;
            break;
        }
        stack.push(res);
      } else if (token.type === 'bool-operator') {
        const right = stack.pop();
        const left = stack.pop();
        let res = false;

        switch (token.value) {
          case 'and':
            res = left && right;
            break;
          case 'or':
            res = left || right;
            break;
        }
        stack.push(res);
      } else if (token.type === 'label') {
        // operand
        let value = this._data[token.value];
        if (value === undefined) {
          stack.push(null);
        } else {
          stack.push(value);
        }
      } else if (token.type === 'number' || token.type === 'size') {
        stack.push(parseInt(token.value, 10));
      } else {
        console.error(`Unknown token type in filter: ${token.type}`);
      }
    }
    return stack.pop();
  }
}

function humanizeService(port, protocol) {
  if (!port || !protocol) {
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

function humanizeSize(size) {
  var i = Math.floor(Math.log(size) / Math.log(1000));
  return (size / Math.pow(1000, i)).toFixed(2) + ' ' + ['B', 'KB', 'MB', 'GB', 'TB'][i];
}