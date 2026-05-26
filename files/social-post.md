# LinkedIn Post

---

Been heads-down building FortressWAF for a while. Just cut v1.4.0 and honestly, I think it's starting to look like something serious.

Here's what we added:

- **JA3 fingerprinting** — parses real TLS ClientHello, computes MD5 hash, matches known bad actors. No more relying on IP alone.
- **Behavioral engine** — per-IP velocity tracking, path entropy analysis, reputation scoring with auto-escalation to block.
- **WASM runtime** — hot-loadable .wasm modules via wazero. Write your own inspector in any language that compiles to WASM, drop it in a folder, it just works.
- **HTTP desync detection** — CL.TE, TE.CL, obs-fold, chunked encoding abuse. Request smuggling is still alive and well.
- **Adaptive challenge system** — graduated response from JS challenge → CAPTCHA → tarpit → block, based on cumulative threat score.
- **eBPF telemetry** — Linux packet-level observability. Still early but the foundation is there.

Also shipped parser hardening (unicode bypass, normalization attacks, HTTP downgrade detection), a false positive reduction engine with shadow mode + learning mode, and performance isolation with circuit breakers per inspector.

Pure Go, single binary, no heavy dependencies.

**We're looking for people who want to help:**
- Go backend folks
- WASM runtime / sandbox people
- eBPF / kernel engineers
- Frontend (React / dashboard)
- Security researchers who want to contribute rules
- Docs / DevRel

If any of this sounds interesting, hop on the repo or DM me. PRs genuinely welcome.

https://github.com/FortressWAF/FortressWAF

#OpenSource #WAF #Cybersecurity #GoLang

---

# X (Twitter) Post

---

just dropped fortresswaf v1.4.0

everything new:
• ja3 fingerprinting — real clienthello parser, md5 hash, known-bad matching
• behavioral ip rep + velocity + path entropy
• wasm hot-loadable inspectors (wazero runtime)
• http desync / smuggling (cl.te, te.cl, obs-fold)
• adaptive challenge: js → captcha → tarpit → block
• ebpf telemetry (linux, early but working)

plus parser hardening, fp reduction engine, shadow mode, learning mode, circuit breakers.

pure go. single binary. no bloat.

want to contribute? come hang out on the repo. need go folks, wasm people, ebpf nerds, security researchers.

github.com/FortressWAF/FortressWAF

#golang #waf #infosec

---

# Instagram Post

Caption:

---

we just dropped fortresswaf v1.4.0 — 6 new detection modules, all in pure go

ja3 fingerprinting · behavioral engine · wasm hot-loadable inspectors · http desync / smuggling detection · adaptive challenge system (js→captcha→tarpit→block) · ebpf telemetry

plus parser hardening, false positive reduction, circuit breakers. single binary, no bloat.

we need contributors. go, wasm, ebpf, security research, frontend. come build with us.

link in bio / github.com/FortressWAF/FortressWAF

---

**Stories / Reels text overlay ideas:**

Frame 1: "FortressWAF v1.4.0 just dropped"
Frame 2: "6 new detection modules"
Frame 3: "JA3 · Behavioral · WASM · Desync · Adaptive · eBPF"
Frame 4: "Pure Go. Open Source. We need you."
Frame 5: "github.com/FortressWAF/FortressWAF"

---

# TikTok Post

Script / voiceover:

---

**Visual: code editor scrolling through Go files**

"we just shipped fortresswaf v1.4.0. six new detection modules in pure go. no python. no node. just a single binary."

**Visual: terminal with go build running**

"ja3 fingerprinting that parses real tls clienthello. behavioral engine with ip velocity and path entropy. hot-loadable wasm inspectors — write your own detection logic and drop it in a folder."

**Visual: architecture diagram or demo**

"http desync smuggling detection. adaptive challenge that goes from js captcha all the way to tarpit. ebpf telemetry on linux."

**Visual: github repo page**

"plus parser hardening, false positive reduction, circuit breakers. we're open source and looking for contributors. go devs, wasm people, ebpf nerds, security researchers."

**Visual: link on screen**

"github.com/FortressWAF/FortressWAF"

---
