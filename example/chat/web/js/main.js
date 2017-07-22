var Connection = new Client("ws://localhost/8080/ws");

var Messages = {
	list: [],
	init: function() {},
	send: function(msg) {
		Messages.list.push(msg)
	}
};

var User = {
	name: "gopher"
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
					m(".row", [
						m(".col-xs-7", [
							m("ul.nav.nav-pills", [
								m("li.nav-header", {role: "presentation"}, [
									m("a#chat", { href: "/chat", oncreate: m.route.link }, "GoChat"),
								]),
								nav("/about", "about"),
							]),
						]),
						m(".col-xs-5.text-right", [
							m("form.user", {
								onsubmit: function(e) {
									e.preventDefault()
									if (User.name.length != 0) {
										Bootstrap.login()
									}
								}	
							}, [
								m("span.glyphicon.glyphicon-user"),
								m("span.user-prefix", "@"),
								m("input.user-name", {
									value: User.name,
									oncreate: function(vnode) {
										$(vnode.dom).css('width', textWidth(vnode.dom.value));
									},
									onfocus: function(e) {
										var prev = this.value
										setTimeout(function() {
											e.target.value = prev
										}, 1)
									},
									oninput: function(e) {
										var el = e.target
										User.name = el.value
										$(el).css('width', textWidth(el.value));
									},
									onchange: function(e) { 
										if (User.name.length != 0) {
											Bootstrap.login()
										}
									}
								}),
							])
						])
					])
				])
			]),
			m("div.container.content", [
				m("section", vnode.children)
			])
		]
	}
};

function textWidth(text) {
	var ret = 0;
	var temp = document.getElementById("temp");
	m.render(temp, m("div", {
		oncreate: function(vnode) {
			ret = vnode.dom.clientWidth;
		},
		onupdate: function(vnode) {
			ret = vnode.dom.clientWidth;
		},
		style: {
			"font-weight": "500",
			"font-size":   "14px",
			"position":    "absolute",
			"visibility":  "hidden",
			"height":      "auto",
			"width":       "auto",
			"white-space": "nowrap"
		},
	}, text))
	return ret + 5 + "px"
}

var Message = {
	text: "",
	reset: function() {
		var text = Message.text
		Message.text = ""
		return text
	}
}

var Compose = {
	oncreate: function() {
		setTimeout(function() {
			document.getElementById("compose").focus()
		}, 10)
	},
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
								m("input.form-control.compose-input#compose", {
									type: "text",
									value: Message.text,
									placeholder: "Write a message...",
									autocomplete: "off",
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
	login: function() {
		var self = this;
		this.ready = false

		m.route.set("/")

		return Connection.call("hello", Object.assign({}, User))
			.then(function(user) {
				Object.assign(User, user)
				self.ready = true;
			})
			.catch(function(err) {
				console.warn("call hello error:", err);
				self.err = err;
			})
			.then(function() {
				Bootstrap.spinner.stop()
				m.redraw();
			});
	},
	oncreate: function() {
		if (this.ready) {
			return
		}
		if (this.err != null) {
			return;
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
		if (this.ready) {
			m.route.set("/chat")
			return
		}
		if (this.err != null) {
			return m(".crash", [
				m(".crash-message", "Oh snap! Something went wrong! =(")
			])
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
