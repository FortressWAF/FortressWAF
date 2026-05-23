import os
import re
import pickle
import logging
from typing import Dict, Optional, Tuple, List
from ipaddress import ip_address, IPv4Address, IPv6Address

import numpy as np
from sklearn.ensemble import RandomForestClassifier

from training.feature_engineering import compute_entropy

logger = logging.getLogger(__name__)

MODEL_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), "models", "persisted")
os.makedirs(MODEL_DIR, exist_ok=True)

BOT_MODEL_PATH = os.path.join(MODEL_DIR, "bot_model.pkl")

KNOWN_GOOD_BOTS = {
    "googlebot", "bingbot", "slurp", "duckduckbot", "baiduspider",
    "yandexbot", "facebookexternalhit", "twitterbot", "linkedinbot",
    "applebot", "ia_archiver", "msnbot", "exabot", "facebot",
    "slackbot", "telegrambot", "whatsapp", "discordbot",
    "google-structured-data-testing-tool", "w3c_validator",
    "validator", "feedfetcher-google", "google-proxy-message",
    "adsbot-google", "mediapartners-google", "apache-httpclient",
    "python-requests", "curl", "wget",
}

KNOWN_BAD_BOTS = [
    r"masscan", r"zgrab", r"nmap", r"nikto", r"nessus", r"openvas",
    r"acunetix", r"netsparker", r"appscan", r"burp", r"zap",
    r"sqlmap", r"havij", r"sqlninja", r"w3af", r"wapiti",
    r"arachni", r"vega", r"wpscan", r"joomscan", r"droopescan",
    r"dirbuster", r"gobuster", r"ffuf", r"wfuzz", r"wfuzz",
    r"aircrack", r"hydra", r"medusa", r"john", r"hashcat",
    r"python[-_]?requests", r"python[-_]?urllib", r"python[-_]?httplib",
    r"okhttp", r"ruby[-_]?http", r"php[-_]?curl",
    r"curl/", r"wget/", r"libcurl", r"perl[-_]?libwww",
    r"node[-_]?fetch", r"axios", r"got[^-]", r"superagent",
    r"scrapy", r"mechanize", r"selenium", r"playwright",
    r"puppeteer", r"phantomjs", r"headless", r"htmlunit",
    r"cyberdog", r"screamingfrog", r"mj12bot", r"semrushbot",
    r"ahrefsbot", r"majestic", r"rogerbot", r"dotbot",
    r"spinn3r", r"blexbot", r"webtrends", r"cognitiveseo",
    r"petalbot", r"mojeekbot", r"aiohttp", r"httpx",
    r"go-http-client", r"container-app", r"internetmeasurement",
    r"zgrab", r"custom-http-client", r"cloudstack",
    r"wrk", r"ab", r"http_load", r"siege", r"vegeta",
    r"hey", r"boom", r"bombardier",
]

GOOD_BOT_DOMAINS = {
    "googlebot.com", "google.com",
    "search.msn.com", "msn.com",
    "crawl.yahoo.net", "yahoo.net",
    "crawl.baidu.com", "baidu.com",
    "yandex.com", "yandex.ru",
    "facebook.com", "fb.com",
    "twitter.com", "twttr.com",
    "linkedin.com",
    "duckduckgo.com",
    "apple.com",
    "slack.com",
    "discord.com",
    "telegram.org",
}


def _standardize_ua(ua: str) -> str:
    return re.sub(r"[/\s()]+", " ", ua.lower().strip())


def _is_known_good_bot(ua: str) -> bool:
    ua_lower = ua.lower()
    for bot in KNOWN_GOOD_BOTS:
        if bot in ua_lower:
            return True
    return False


def _is_known_bad_bot(ua: str) -> Tuple[bool, str]:
    ua_lower = ua.lower()
    for pattern in KNOWN_BAD_BOTS:
        if re.search(pattern, ua_lower):
            matched = pattern.replace("\\", "").rstrip("?+*")
            return True, matched
    return False, "unknown"


def _detect_headless_browser(ua: str, headers: Dict[str, str]) -> bool:
    ua_lower = ua.lower()
    headless_patterns = [
        "headless", "phantomjs", "htmlunit", "puppeteer", "playwright",
        "selenium", "webdriver",
    ]
    if any(p in ua_lower for p in headless_patterns):
        return True

    header_flags = [
        "headless" in headers.get("sec-ch-ua", "").lower(),
        headers.get("sec-ch-ua-platform", "") == "",
        "webdriver" in headers.get("sec-ch-ua", "").lower(),
    ]
    return any(header_flags)


class BotDetector:
    def __init__(self, random_state: int = 42):
        self.random_state = random_state
        self.model: Optional[RandomForestClassifier] = None
        self.load_model()

    def _extract_features(self, user_agent: str, headers: Dict[str, str],
                          request_timing_ms: int, accept_language: str,
                          js_challenge_result: Optional[bool]) -> np.ndarray:
        features = []
        ua = _standardize_ua(user_agent)

        features.append(len(user_agent))

        ua_entropy = compute_entropy(user_agent)
        features.append(ua_entropy)

        features.append(request_timing_ms)

        lang_parts = accept_language.split(",")
        features.append(len(lang_parts))
        q_values = [re.search(r"q=([\d.]+)", p) for p in lang_parts]
        avg_q = np.mean([float(m.group(1)) for m in q_values if m]) if any(q_values) else 1.0
        features.append(avg_q)

        sec_ch_ua = headers.get("sec-ch-ua", "")
        features.append(len(sec_ch_ua))

        features.append(1 if "sec-ch-ua-mobile" in headers else 0)
        features.append(1 if "sec-ch-ua-platform" in headers else 0)
        features.append(1 if "sec-ch-ua-arch" in headers else 0)
        features.append(1 if "sec-ch-ua-bitness" in headers else 0)
        features.append(1 if "sec-ch-ua-model" in headers else 0)
        features.append(1 if "sec-ch-ua-full-version" in headers else 0)

        header_order = [k.lower() for k in headers.keys()]
        browser_headers = {"host", "user-agent", "accept", "accept-language",
                           "accept-encoding", "connection", "upgrade-insecure-requests",
                           "sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site",
                           "sec-fetch-user", "dnt", "referer", "origin"}
        header_anomaly = sum(1 for h in header_order if h not in browser_headers and not h.startswith("sec-ch-"))
        features.append(header_anomaly)

        features.append(1 if _detect_headless_browser(user_agent, headers) else 0)

        is_known_good = _is_known_good_bot(user_agent)
        features.append(1 if is_known_good else 0)

        is_bad, _ = _is_known_bad_bot(user_agent)
        features.append(1 if is_bad else 0)

        features.append(1 if js_challenge_result is False else 0)
        features.append(1 if js_challenge_result is True else 0)

        features.append(ua.count("/"))
        features.append(ua.count("("))
        features.append(1 if ua.startswith("mozilla") else 0)
        features.append(1 if "chrome" in ua or "safari" in ua or "firefox" in ua or "edge" in ua else 0)
        features.append(1 if "mobile" in ua else 0)
        features.append(1 if "bot" in ua else 0)
        features.append(1 if "crawler" in ua else 0)
        features.append(1 if "spider" in ua else 0)
        features.append(1 if "scanner" in ua else 0)
        features.append(1 if "checker" in ua else 0)
        features.append(1 if "scrape" in ua else 0)
        features.append(1 if "wordpress" in ua or "wp" in ua else 0)

        accept_header = headers.get("accept", "")
        features.append(0 if accept_header == "*/*" else 1 if accept_header else 0.5)

        encoding = headers.get("accept-encoding", "")
        features.append(len(encoding))

        return np.array(features, dtype=np.float64)

    def train(self, features_batch: np.ndarray, labels: np.ndarray):
        self.model = RandomForestClassifier(
            n_estimators=150,
            max_depth=20,
            min_samples_split=5,
            min_samples_leaf=2,
            class_weight="balanced",
            random_state=self.random_state,
            n_jobs=-1,
        )
        self.model.fit(features_batch, labels)
        self.save_model()

    def partial_fit(self, features_batch: np.ndarray, labels: np.ndarray):
        if self.model is None:
            self.train(features_batch, labels)
        else:
            n_estimators = min(self.model.n_estimators + 20, 300)
            self.model.n_estimators = n_estimators
            self.model.fit(
                np.vstack([self.model.estimators_[0].tree_.__getattribute__("") if False else features_batch]),
                labels,
            )
            self.save_model()

    def score(self, user_agent: str, headers: Dict[str, str],
              request_timing_ms: int, accept_language: str,
              js_challenge_result: Optional[bool] = None) -> float:
        features = self._extract_features(user_agent, headers, request_timing_ms,
                                          accept_language, js_challenge_result).reshape(1, -1)

        if self.model is None:
            self.model = RandomForestClassifier(
                n_estimators=100, random_state=self.random_state, n_jobs=-1,
            )
            dummy_X = np.random.randn(10, features.shape[1])
            dummy_y = np.array([0, 0, 0, 0, 0, 1, 1, 1, 1, 1])
            self.model.fit(dummy_X, dummy_y)

        proba = self.model.predict_proba(features)[0]
        if proba.shape[0] == 1:
            return float(proba[0])
        return float(proba[1])

    def classify(self, user_agent: str, headers: Dict[str, str],
                 request_timing_ms: int, accept_language: str,
                 js_challenge_result: Optional[bool] = None) -> Tuple[float, bool, str]:
        ua_lower = user_agent.lower()

        if _is_known_good_bot(user_agent):
            return 0.0, True, "known-good"

        is_bad, bad_type = _is_known_bad_bot(user_agent)
        if is_bad:
            return 1.0, True, "known-bad"

        if _detect_headless_browser(user_agent, headers):
            bot_score = self.score(user_agent, headers, request_timing_ms, accept_language, js_challenge_result)
            return max(0.7, bot_score), True, "headless"

        if request_timing_ms < 50:
            return 1.0, True, "known-bad"

        bot_score = self.score(user_agent, headers, request_timing_ms, accept_language, js_challenge_result)
        is_bot = bot_score >= 0.6

        if is_bot:
            if bot_score >= 0.85:
                return bot_score, True, "known-bad"
            return bot_score, True, "unknown"

        return bot_score, False, "unknown"

    def save_model(self):
        if self.model is not None:
            path = BOT_MODEL_PATH
            with open(path, "wb") as f:
                pickle.dump(self.model, f)
            logger.info(f"Bot model saved to {path}")

    def load_model(self):
        path = BOT_MODEL_PATH
        if os.path.exists(path):
            try:
                with open(path, "rb") as f:
                    self.model = pickle.load(f)
                logger.info(f"Bot model loaded from {path}")
            except Exception as e:
                logger.error(f"Failed to load bot model: {e}")
                self.model = None

    @property
    def version(self) -> str:
        return f"random-forest-bot-v2.0"

    @property
    def is_loaded(self) -> bool:
        return self.model is not None
