package engine

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
)

type XSSEngine struct {
	mu             sync.RWMutex
	devMode        bool
	htmlTags       []*regexp.Regexp
	eventHandlers  []*regexp.Regexp
	jsProtocols    []*regexp.Regexp
	polyglot       []*regexp.Regexp
	svgPatterns    []*regexp.Regexp
	cssInjection   []*regexp.Regexp
	encodedXSS     *regexp.Regexp
	reflectedXSS   *regexp.Regexp
	customCSP      string
}

func NewXSSEngine(devMode bool) *XSSEngine {
	e := &XSSEngine{
		devMode: devMode,
	}

	e.compilePatterns()
	e.setupCSP()

	return e
}

func (e *XSSEngine) Name() string { return "xss" }

func (e *XSSEngine) compilePatterns() {
	e.htmlTags = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:<script[^>]*>[^<]*</script>)`),
		regexp.MustCompile(`(?i)(?:<iframe[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<object[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<embed[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<applet[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<meta[^>]*http-equiv[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<link[^>]*href[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<base[^>]*href[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<form[^>]*action[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<img[^>]*onerror[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<body[^>]*onload[^>]*>)`),
		regexp.MustCompile(`(?i)(?:<svg[^>]*/svg>)`),
		regexp.MustCompile(`(?i)(?:<math[^>]*>)`),
	}

	e.eventHandlers = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:onabort|onautocomplete|onautocompleteerror|onblur|oncancel|oncanplay|oncanplaythrough|onchange|onclick|onclose|oncontextmenu|oncuechange|ondblclick|ondrag|ondragend|ondragenter|ondragleave|ondragover|ondragstart|ondrop|ondurationchange|onemptied|onended|onerror|onfocus|onfocusin|onfocusout|ongotpointercapture|oninput|oninvalid|onkeydown|onkeypress|onkeyup|onload|onloadeddata|onloadedmetadata|onloadstart|onlostpointercapture|onmousedown|onmousemove|onmouseout|onmouseover|onmouseup|onmousewheel|onpause|onplay|onplaying|onpointercancel|onpointerdown|onpointerenter|onpointerleave|onpointermove|onpointerout|onpointerover|onpointerup|onprogress|onratechange|onreset|onresize|onscroll|onseeked|onseeking|onselect|onselectionchange|onselectstart|onshow|onstalled|onsubmit|onsuspend|ontimeupdate|ontoggle|onvolumechange|onwaiting|onwheel)`),
		regexp.MustCompile(`(?i)(?:onmouseenter|onmouseleave|onpointerrawupdate|onbeforeinput|onbeforetoggle|oncontentvisibilityautostatechange)`),
		regexp.MustCompile(`(?i)(?:onpageshow|onpagehide|onpopstate|onhashchange|onbeforeunload|onunload)`),
	}

	e.jsProtocols = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:javascript\s*:)`),
		regexp.MustCompile(`(?i)(?:vbscript\s*:)`),
		regexp.MustCompile(`(?i)(?:data\s*:\s*(?:text/html|application/xhtml))`),
		regexp.MustCompile(`(?i)(?:livescript\s*:)`),
		regexp.MustCompile(`(?i)(?:mocha\s*:)`)}

	e.polyglot = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:jaVasCript:[\s\S]*?[<\"'])`),
		regexp.MustCompile(`(?i)(?:\\x22.*onerror\\x3d)`),
		regexp.MustCompile(`(?i)(?:\\x3Cscript\\x3E)`),
		regexp.MustCompile(`(?i)(?:<[^>]*>[\s\S]*?<)`),
	}

	e.svgPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:<svg[^>]*>[\s\S]*?<script)`),
		regexp.MustCompile(`(?i)(?:<svg[^>]*onload)`),
		regexp.MustCompile(`(?i)(?:<svg[^>]*>[\s\S]*?<animate)`),
		regexp.MustCompile(`(?i)(?:<svg[^>]*>[\s\S]*?<set)`),
		regexp.MustCompile(`(?i)(?:<svg[^>]*>[\s\S]*?<use)`),
		regexp.MustCompile(`(?i)(?:<svg[^>]*>[\s\S]*?<desc>)`),
	}

	e.cssInjection = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:expression\s*\()`),
		regexp.MustCompile(`(?i)(?:-moz-binding)`),
		regexp.MustCompile(`(?i)(?:behavior\s*:)`),
		regexp.MustCompile(`(?i)(?:@import\s+url)`),
		regexp.MustCompile(`(?i)(?:url\s*\(\s*['"]?\s*javascript:)`),
		regexp.MustCompile(`(?i)(?:position\s*:\s*fixed)`),
	}

	e.encodedXSS = regexp.MustCompile(`(?i)(?:\\x[0-9a-f]{2}|\\u[0-9a-f]{4}|%[0-9a-f]{2}|&#x?[0-9a-f]+;).*(?:script|alert|prompt|confirm|onerror|onload)`)
}

func (e *XSSEngine) setupCSP() {
	e.customCSP = "default-src 'self'; " +
		"script-src 'self' 'strict-dynamic' 'nonce-{nonce}' 'unsafe-inline' http: https:; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"require-trusted-types-for 'script';"
}

func (e *XSSEngine) Inspect(ctx *RequestContext) (*Decision, error) {
	targets := e.extractTargets(ctx)

	for _, target := range targets {
		if dec := e.inspectValue(target.value, target.source); dec != nil {
			return dec, nil
		}
	}

	return nil, nil
}

type xssTarget struct {
	value  string
	source string
}

func (e *XSSEngine) extractTargets(ctx *RequestContext) []xssTarget {
	var targets []xssTarget
	for k, v := range ctx.QueryParams {
		for _, val := range v {
			targets = append(targets, xssTarget{value: val, source: fmt.Sprintf("query:%s", k)})
		}
	}
	for k, v := range ctx.FormParams {
		for _, val := range v {
			targets = append(targets, xssTarget{value: val, source: fmt.Sprintf("form:%s", k)})
		}
	}
	if ctx.Body != nil {
		targets = append(targets, xssTarget{value: string(ctx.Body), source: "body"})
	}
	for k, v := range ctx.Headers {
		lower := strings.ToLower(k)
		if lower == "referer" || lower == "origin" || lower == "x-forwarded-for" {
			targets = append(targets, xssTarget{value: v, source: fmt.Sprintf("header:%s", k)})
		}
	}
	for k, v := range ctx.Cookies {
		targets = append(targets, xssTarget{value: v, source: fmt.Sprintf("cookie:%s", k)})
	}
	return targets
}

func (e *XSSEngine) inspectValue(value, source string) *Decision {
	if value == "" {
		return nil
	}

	for _, pattern := range e.htmlTags {
		if pattern.MatchString(value) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "XSS001",
				RuleName: "HTML Tag Injection",
				Severity: "critical",
				Score:    90,
				Evidence: fmt.Sprintf("HTML tag injection in %s: %s", source, pattern.String()),
			}
		}
	}

	for _, pattern := range e.eventHandlers {
		if pattern.MatchString(value) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "XSS002",
				RuleName: "Event Handler Injection",
				Severity: "critical",
				Score:    90,
				Evidence: fmt.Sprintf("event handler injection in %s", source),
			}
		}
	}

	for _, pattern := range e.jsProtocols {
		if pattern.MatchString(value) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "XSS003",
				RuleName: "JavaScript Protocol",
				Severity: "critical",
				Score:    85,
				Evidence: fmt.Sprintf("javascript protocol detected in %s", source),
			}
		}
	}

	for _, pattern := range e.polyglot {
		if pattern.MatchString(value) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "XSS004",
				RuleName: "Polyglot XSS Payload",
				Severity: "critical",
				Score:    90,
				Evidence: fmt.Sprintf("polyglot XSS payload in %s", source),
			}
		}
	}

	for _, pattern := range e.svgPatterns {
		if pattern.MatchString(value) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "XSS005",
				RuleName: "SVG Injection",
				Severity: "critical",
				Score:    85,
				Evidence: fmt.Sprintf("SVG injection in %s", source),
			}
		}
	}

	for _, pattern := range e.cssInjection {
		if pattern.MatchString(value) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "XSS006",
				RuleName: "CSS Injection",
				Severity: "high",
				Score:    75,
				Evidence: fmt.Sprintf("CSS injection in %s", source),
			}
		}
	}

	if e.encodedXSS.MatchString(value) {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "XSS007",
			RuleName: "Encoded XSS",
			Severity: "high",
			Score:    75,
			Evidence: fmt.Sprintf("encoded XSS pattern in %s", source),
		}
	}

	return nil
}

func (e *XSSEngine) GenerateCSP(nonce string) string {
	return strings.ReplaceAll(e.customCSP, "{nonce}", nonce)
}

func (e *XSSEngine) ScanResponse(body []byte) *Decision {
	if len(body) == 0 {
		return nil
	}

	bodyStr := string(body)

	for _, pattern := range e.htmlTags {
		if pattern.MatchString(bodyStr) {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "XSS008",
				RuleName: "Reflected XSS",
				Severity: "critical",
				Score:    95,
				Evidence: "reflected XSS detected in response body",
			}
		}
	}

	return nil
}

var _ = slog.Debug
