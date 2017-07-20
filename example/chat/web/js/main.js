var Messages = {
	list: [],
	init: function() {
		// TODO
		if (Messages.list.length == 0) {
		Messages.list.push({
			time:   Date.now(),
			author: Bootstrap.user.name,
			text:   "hello, everybody!",
		})
		}
	},
	send: function(msg) {
		Messages.list.push(msg)
	}
};

function login(name) {
	return new Promise(function(resolve, reject) {
		setTimeout(function() {
			resolve({
				name: name,
			})
		}, 1000)
	})
}

function send(msg) {
	return new Promise(function(resolve, reject) {
		setTimeout(function() {
			resolve()
		}, 1000)
	})
}

var User = {
	name: "gopher"
}

var Login = {
	view: function() {
		return m("form.login",
			{onsubmit: function(e) {
				e.preventDefault()
				if (User.name.length == 0) {
					return
				}
				Bootstrap.login()
			}},
		   	[
				m("input.login-name", {
					value: User.name,
					oninput: m.withAttr("value", function(value) { 
						User.name = value
					})
				}),
				m("button.button[type=submit]", "relogin"),
			]
		)
	}
}


var Chat = {
	oninit: function() {
		Messages.init()
	},
	view: function() {
		return m(".messages", [
			Messages.list.map(function(msg) {
				msg = Object.assign({
					time:   Date.now(),
					author: "@_@",
				}, msg)

				var d = new Date(msg.time)
				return m("p.message", [
					m("span.message-time",   d.toLocaleTimeString()),
					m("span.message-author", msg.author + ">"),
					m("span.message-text",   msg.text)
				])
			})
		])
	}
};

var App = {
	view: function(vnode) {
		return m("app", [
			m("nav", [
				m("a.nav-item", {href: "/chat", oncreate: m.route.link}, "chat"),
				m("a.nav-item", {href: "/about", oncreate: m.route.link}, "about"),
				m("a.nav-item", {href: "/login", oncreate: m.route.link}, "login")
			]),
			m("section", vnode.children)
		])
	}
};

var Message = {
	text: "",
	reset: function() {
		var text = Message.text
		Message.text = ""
		return text
	}
}

var Compose = {
	view: function() {
		return m("form.compose",
			{onsubmit: function(e) {
				e.preventDefault()
				var text = Message.reset()
				if (text.length == 0) {
					return
				}
				Messages.send({
					author: Bootstrap.user.name,
					text:   text,
					time:   Date.now()
				})
			}},
		   	[
				m("input", {
					value: Message.text,
					oninput: m.withAttr("value", function(value) { 
						Message.text = value
					})
				}),
				m("button.button[type=submit]", "send"),
			]
		)
	}
}

var About = {
	view: function() {
		return m("div", "hello, websocket!")
	}
};

var Bootstrap = {
	ready: false,
	user:  {},
	login: function() {
		Bootstrap.ready = false
		m.route.set("/")
		return login(User)
			.then(function(ws) {
				Bootstrap.ready = true
				Bootstrap.user  = Object.assign({}, User)
				m.redraw()
			})
	},
	oncreate: function() {
		if (Bootstrap.ready) {
			return
		}
		return Bootstrap.login()
	},
	view: function(vnode) {
		if (vnode.state.ready) {
			m.route.set("/chat")
			return
		}
		return m("div", "loading...")
	}
}

m.route(document.body, "/", {
	"/": {
		render: function() {
			return m(Bootstrap)
		},
	},
	"/login": {
		render: function() {
			return m(App, m(Login))
		}	
	},
	"/chat": {
		render: function() {
			if (!Bootstrap.ready) {
				m.route.set("/")
				return
			}
			return m(App, m(Chat), m(Compose))
		}	
	},
    "/about": {
		render: function() {
			return m(App, m(About))
		}	
	}
});
