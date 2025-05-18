class WebsocketConnection {
    constructor(handler) {
        const url = new URL("/packets", location.href);
        if (url.protocol === 'http:') {
            url.protocol = 'ws';
        } else if (url.protocol === 'https:') {
            url.protocol = 'wss';
        }
        this.url = url;

        this.pause = false;
        this.ws = null;
        this.handler = handler;
    }

    connect() {
        console.log('Connecting to WebSocket server at', this.url.href);
        this.pause = false;
        this.ws = new WebSocket(this.url);
        const ws = this.ws;

        ws.onmessage = (e) => {
            if (this.handler) {
                this.handler(e);
            }
        };

        ws.onclose = (e) => {
            if (this.pause) {
                return;
            }

            console.log('Socket is closed. Reconnect will be attempted in 1 second.', e.reason);
            setTimeout(() => {
                this.connect();
            }, 1000);
        };

        ws.onerror = (err) => {
            console.error('Socket encountered error: ', err.message, 'Closing socket');
            ws.close();
        };
    }

    disconnect() {
        this.pause = true;
        if (this.ws) {
            this.ws.close();
        }
    }

    toggleConnect() {
        if (this.pause) {
            this.connect();
        } else {
            this.disconnect();
        }

        return this.pause;
    }
}