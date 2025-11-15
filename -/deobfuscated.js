(() => {
  "use strict";

  var e = {};
  var t = {};
  function r(o) {
    var n = t[o];
    if (n !== undefined) {
      return n.exports;
    }
    var a = t[o] = {
      id: o,
      loaded: false,
      exports: {}
    };
    var i = true;
    try {
      e[o].call(a.exports, a, a.exports, r);
      i = false;
    } finally {
      if (i) {
        delete t[o];
      }
    }
    a.loaded = true;
    return a.exports;
  }
  r.m = e;
  (() => {
    var e = [];
    r.O = (t, o, n, a) => {
      if (o) {
        a = a || 0;
        for (var i = e.length; i > 0 && e[i - 1][2] > a; i--) {
          e[i] = e[i - 1];
        }
        e[i] = [o, n, a];
        return;
      }
      var d = Infinity;
      for (var i = 0; i < e.length; i++) {
        for (var [o, n, a] = e[i], c = true, l = 0; l < o.length; l++) {
          if ((a & false || d >= a) && Object.keys(r.O).every(e => r.O[e](o[l]))) {
            o.splice(l--, 1);
          } else {
            c = false;
            if (a < d) {
              d = a;
            }
          }
        }
        if (c) {
          e.splice(i--, 1);
          var u = n();
          if (u !== undefined) {
            t = u;
          }
        }
      }
      return t;
    };
  })();
  r.n = e => {
    var t = e && e.__esModule ? () => e.default : () => e;
    r.d(t, {
      a: t
    });
    return t;
  };
  (() => {
    var e;
    var t = Object.getPrototypeOf ? e => Object.getPrototypeOf(e) : e => e.__proto__;
    r.t = function (o, n) {
      if (n & 1) {
        o = this(o);
      }
      if (n & 8 || typeof o == "object" && o && (n & 4 && o.__esModule || n & 16 && typeof o.then == "function")) {
        return o;
      }
      var a = Object.create(null);
      r.r(a);
      var i = {};
      e = e || [null, t({}), t([]), t(t)];
      for (var d = n & 2 && o; typeof d == "object" && !~e.indexOf(d); d = t(d)) {
        Object.getOwnPropertyNames(d).forEach(e => i[e] = () => o[e]);
      }
      i.default = () => o;
      r.d(a, i);
      return a;
    };
  })();
  r.d = (e, t) => {
    for (var o in t) {
      if (r.o(t, o) && !r.o(e, o)) {
        Object.defineProperty(e, o, {
          enumerable: true,
          get: t[o]
        });
      }
    }
  };
  r.f = {};
  r.e = e => Promise.all(Object.keys(r.f).reduce((t, o) => {
    r.f[o](e, t);
    return t;
  }, []));
  r.u = e => "static/chunks/" + ({
    2346: "273acdc0",
    2669: "c132bf7d",
    5194: "badf541d",
    5951: "39fb572b"
  }[e] || e) + "." + {
    39: "cb61ebbe8e6a1a34",
    184: "8644f9cad178c9f9",
    985: "fceca228caf8c36c",
    2346: "7c2404d2364fc300",
    2669: "36f177b670dba90f",
    5194: "1811582b9180b4a2",
    5951: "67dfcb26a998d62c",
    7267: "80df54422f59dda0"
  }[e] + ".js";
  r.miniCssF = e => {};
  r.g = function () {
    if (typeof globalThis == "object") {
      return globalThis;
    }
    try {
      return this || Function("return this")();
    } catch (e) {
      if (typeof window == "object") {
        return window;
      }
    }
  }();
  r.o = (e, t) => Object.prototype.hasOwnProperty.call(e, t);
  (() => {
    var e = {};
    var t = "_N_E:";
    r.l = (o, n, a, i) => {
      if (e[o]) {
        e[o].push(n);
        return;
      }
      if (a !== undefined) {
        var d;
        var c;
        for (var l = document.getElementsByTagName("script"), u = 0; u < l.length; u++) {
          var f = l[u];
          if (f.getAttribute("src") == o || f.getAttribute("data-webpack") == t + a) {
            d = f;
            break;
          }
        }
      }
      if (!d) {
        c = true;
        (d = document.createElement("script")).charset = "utf-8";
        d.timeout = 120;
        if (r.nc) {
          d.setAttribute("nonce", r.nc);
        }
        d.setAttribute("data-webpack", t + a);
        d.src = r.tu(o);
      }
      e[o] = [n];
      var s = (t, r) => {
        d.onerror = d.onload = null;
        clearTimeout(p);
        var n = e[o];
        delete e[o];
        if (d.parentNode) {
          d.parentNode.removeChild(d);
        }
        if (n) {
          n.forEach(e => e(r));
        }
        if (t) {
          return t(r);
        }
      };
      var p = setTimeout(s.bind(null, undefined, {
        type: "timeout",
        target: d
      }), 120000);
      d.onerror = s.bind(null, d.onerror);
      d.onload = s.bind(null, d.onload);
      if (c) {
        document.head.appendChild(d);
      }
    };
  })();
  r.r = e => {
    if (typeof Symbol != "undefined" && Symbol.toStringTag) {
      Object.defineProperty(e, Symbol.toStringTag, {
        value: "Module"
      });
    }
    Object.defineProperty(e, "__esModule", {
      value: true
    });
  };
  r.nmd = e => {
    e.paths = [];
    e.children ||= [];
    return e;
  };
  (() => {
    var e;
    r.tt = () => {
      if (e === undefined) {
        e = {
          createScriptURL: e => e
        };
        if (typeof trustedTypes != "undefined" && trustedTypes.createPolicy) {
          e = trustedTypes.createPolicy("nextjs#bundler", e);
        }
      }
      return e;
    };
  })();
  r.tu = e => r.tt().createScriptURL(e);
  r.p = "/_next/";
  (() => {
    var e = {
      8068: 0,
      9070: 0
    };
    r.f.j = (t, o) => {
      var n = r.o(e, t) ? e[t] : undefined;
      if (n !== 0) {
        if (n) {
          o.push(n[2]);
        } else if (/^(8068|9070)$/.test(t)) {
          e[t] = 0;
        } else {
          var a = new Promise((r, o) => n = e[t] = [r, o]);
          o.push(n[2] = a);
          var i = r.p + r.u(t);
          var d = Error();
          r.l(i, o => {
            if (r.o(e, t) && ((n = e[t]) !== 0 && (e[t] = undefined), n)) {
              var a = o && (o.type === "load" ? "missing" : o.type);
              var i = o && o.target && o.target.src;
              d.message = "Loading chunk " + t + " failed.\n(" + a + ": " + i + ")";
              d.name = "ChunkLoadError";
              d.type = a;
              d.request = i;
              n[1](d);
            }
          }, "chunk-" + t, t);
        }
      }
    };
    r.O.j = t => e[t] === 0;
    var t = (t, o) => {
      var n;
      var a;
      var [i, d, c] = o;
      var l = 0;
      if (i.some(t => e[t] !== 0)) {
        for (n in d) {
          if (r.o(d, n)) {
            r.m[n] = d[n];
          }
        }
        if (c) {
          var u = c(r);
        }
      }
      for (t && t(o); l < i.length; l++) {
        a = i[l];
        if (r.o(e, a) && e[a]) {
          e[a][0]();
        }
        e[a] = 0;
      }
      return r.O(u);
    };
    var o = self.webpackChunk_N_E = self.webpackChunk_N_E || [];
    o.forEach(t.bind(null, 0));
    o.push = t.bind(null, o.push.bind(o));
  })();
  r.nc = undefined;
})();