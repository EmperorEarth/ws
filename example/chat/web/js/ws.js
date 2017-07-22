class Client {
	constructor(endpoint) {
		this.endpoint = endpoint;
		this.seq = 0;
		this.ready = false;
		this.ws = null;
		this.pending = {};
	}

	connect() {
		var self = this;
		if (this.ws != null) {
			return this.ws
		}
		return this.ws = new Promise(function(resolve, reject) {
			var ws = new WebSocket(self.endpoint)
			var pending = true;
			ws.onerror = function(err) {
				if (pending) {
					pending = false;
					reject(err)
					return
				}

				console.warn("websocket lifetime error:" + err)
				Object.keys(this.pending).forEach(function(k) {
					this.pending[k].reject(err);
					delete this.pending[k];
				})
			};
			ws.onopen = function() {
				if (pending) {
					pending = false
					resolve(ws)
				}
			};
			ws.onmessage = function(s) {
				console.log(s);

				var msg, dfd;
				try {
					msg = JSON.parse(s);
				} catch (err) {
					console.warn("parse incoming message error:", err);
					return
				}
				dfd = this.pending[msg.id];
				if (dfd == null) {
					console.warn("unknown message id:", msg.id);
					return
				}

				delete this.pending[msg.id];

				if (msg.error != null) {
					dfd.reject(msg.error);
					return;
				}

				dfd.resolve(msg.result);

				return;
			};
		})
	}

	call(method, params) {
		return this.connect()
			.then(function(conn) {
				var seq = this.seq++;
				var dfd = defer();
				this.pending[seq] = dfd;
				conn.send(JSON.stringify({
					id:     seq,
					method: method,
					params: params
				}))
				return dfd.promise;
			})
	}
}

function defer() {
	var d = {}
	d.promise = new Promise(function(resolve, reject) {
		d.resolve = resolve;
		d.reject = reject;
	})
	return d
}
