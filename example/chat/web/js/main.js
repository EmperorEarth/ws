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
	lastScroll: 0,
	oninit: function() {
		Messages.init()
	},
	onupdate: function() {
		var scroll = $(".messages").get(0).scrollHeight
		if (Chat.lastScroll == scroll) {
			return
		}
		Chat.lastScroll = scroll

		$("body").animate({
			scrollTop: scroll
		}, 1000);
	},
	view: function() {
		return m("div.messages", [
			Messages.list.map(function(msg) {
				msg = Object.assign({
					time:   Date.now(),
					author: "@_@",
				}, msg)

				var d = new Date(msg.time)
				return m("p.message", [
					m("span.message-time",   d.toLocaleTimeString()),
					m("span.message-author", msg.author),
					m("span.message-invite", ">"),
					m("span.message-text",   msg.text)
				])
			})
		])
	}
};

var App = {
	view: function(vnode) {
		var nav = function(route, caption) {
			var p = {
				role: "presentation",
			}
			if (m.route.get() == route) {
				p.className = "disabled"
			}
			return m("li", p, [ 
				m("a", { href: route, oncreate: m.route.link }, caption) 
			])
		}
		return [
			m("header.header", [
				m("div.container", [
					m("ul.nav.nav-pills", [
						m("li.nav-header", {role: "presentation"}, "The Gophers Chat"),
						nav("/chat", "chat"),
						nav("/about", "about"),
						nav("/login", "login")
					])
				])
			]),
			m("div.container.content", [
				m("section", vnode.children)
			])
		]
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
		return m("footer.footer", [
			m("div.container", [
				m("form.form-horizontal.compose",
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
						m("div.form-group", [
							m("div.col-xs-12", [
								m("input.form-control.compose-input", {
									type: "text",
									value: Message.text,
									placeholder: "Write a message...",
									oninput: m.withAttr("value", function(value) { 
										Message.text = value
									})
								})
							])
						]),
					]
				)
			])
		])
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
	onremove: function() {
		Bootstrap.spinner.stop()
	},
	oncreate: function() {
		if (Bootstrap.ready) {
			return
		}
		var opts = {
			lines:   17,
			length:  12,
			width:   2,
			radius:  12,
			color:   '#268bd2',
			opacity: 0.1,
			speed:   1.5,
		}
		Bootstrap.spinner = new Spinner(opts).spin(document.body)
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
			return [ m(App, m(Chat)), m(Compose) ]
		}	
	},
    "/about": {
		render: function() {
			return m(App, m(About))
		}	
	}
});
