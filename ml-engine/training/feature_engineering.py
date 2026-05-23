import re
import math
import hashlib
from typing import Dict, List, Optional, Tuple
from collections import Counter

import numpy as np
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.preprocessing import StandardScaler


def extract_http_features(
    method: str,
    path: str,
    headers: Dict[str, str],
    body: Optional[str] = None,
    query_params: Optional[Dict[str, str]] = None,
) -> np.ndarray:
    features = []

    method_encoded = {"GET": 0, "POST": 1, "PUT": 2, "DELETE": 3, "PATCH": 4, "HEAD": 5, "OPTIONS": 6, "CONNECT": 7, "TRACE": 8}
    features.append(method_encoded.get(method.upper(), -1))

    features.append(len(path))
    features.append(path.count("/"))
    features.append(path.count("."))
    features.append(path.count(".."))
    features.append(path.count("//"))

    header_count = len(headers)
    features.append(header_count)

    header_names_lower = [k.lower() for k in headers.keys()]
    expected_headers = {"host", "user-agent", "accept", "content-type", "content-length", "connection", "referer", "cookie", "authorization", "accept-encoding", "accept-language", "origin", "x-forwarded-for", "cache-control", "x-requested-with"}
    header_anomaly_score = sum(1 for h in header_names_lower if h not in expected_headers)
    features.append(header_anomaly_score)

    body_size = len(body) if body else 0
    features.append(body_size)

    query_param_count = len(query_params) if query_params else 0
    features.append(query_param_count)

    combined_text = (path or "") + " " + (body or "") + " " + " ".join(f"{k}={v}" for k, v in (query_params or {}).items())

    special_chars = r"[<>\"'%;()&+$`|\\{}\[\]!@#]"
    special_char_ratio = len(re.findall(special_chars, combined_text)) / max(len(combined_text), 1)
    features.append(special_char_ratio)

    numeric_ratio = len(re.findall(r"\d", combined_text)) / max(len(combined_text), 1)
    features.append(numeric_ratio)

    uppercase_ratio = len(re.findall(r"[A-Z]", combined_text)) / max(len(combined_text), 1)
    features.append(uppercase_ratio)

    features.append(compute_entropy(combined_text))

    null_bytes = combined_text.count("\x00")
    features.append(null_bytes)

    features.append(combined_text.count("\\x"))

    features.append(combined_text.count("union"))
    features.append(combined_text.count("select"))
    features.append(combined_text.count("script"))
    features.append(combined_text.count("onerror"))
    features.append(combined_text.count("onload"))
    features.append(combined_text.count("alert("))
    features.append(combined_text.count("eval("))
    features.append(combined_text.count("exec("))
    features.append(combined_text.count("system("))
    features.append(combined_text.count("../../"))
    features.append(combined_text.count("....//"))
    features.append(combined_text.count("${"))
    features.append(combined_text.count("{{"))
    features.append(combined_text.count("%00"))
    features.append(combined_text.count("base64"))

    return np.array(features, dtype=np.float64)


def compute_entropy(text: str) -> float:
    if not text:
        return 0.0
    byte_data = text.encode("utf-8", errors="replace")
    counts = Counter(byte_data)
    total = len(byte_data)
    entropy = -sum((count / total) * math.log2(count / total) for count in counts.values())
    return entropy


def tokenize_payload(payload: str) -> List[str]:
    payload = payload.lower()

    tokens = re.split(r"[^a-zA-Z0-9_\-]+", payload)
    tokens = [t for t in tokens if len(t) > 1]

    sql_keywords = ["select", "from", "where", "union", "insert", "update", "delete", "drop", "alter",
                    "create", "or", "and", "not", "in", "like", "between", "having", "group", "order",
                    "by", "limit", "offset", "as", "join", "on", "null", "true", "false"]
    script_tokens = ["script", "alert", "onerror", "onload", "onclick", "onmouseover", "onfocus",
                     "onblur", "onsubmit", "onchange", "img", "src", "iframe", "document", "cookie",
                     "window", "fetch", "xmlhttprequest", "constructor", "prototype"]
    path_tokens = ["etc", "passwd", "bin", "sh", "bash", "cmd", "windows", "system32", "boot",
                   "ini", "config", "admin", "backup", "wp-config", "env"]

    special_tokens = [t for t in payload.split() if t in sql_keywords or t in script_tokens or t in path_tokens]
    tokens.extend(special_tokens)

    return tokens


def generate_ngrams(tokens: List[str], n: int = 3) -> List[str]:
    ngrams = []
    for i in range(len(tokens) - n + 1):
        ngrams.append(" ".join(tokens[i:i + n]))
    return ngrams


def build_tfidf_vectorizer(corpus: List[str], max_features: int = 5000) -> TfidfVectorizer:
    vectorizer = TfidfVectorizer(
        max_features=max_features,
        analyzer="word",
        token_pattern=r"(?u)\b\w+\b",
        ngram_range=(1, 3),
        sublinear_tf=True,
        min_df=2,
        max_df=0.95,
    )
    vectorizer.fit(corpus)
    return vectorizer


def normalize_features(features: np.ndarray, scaler: Optional[StandardScaler] = None) -> Tuple[np.ndarray, StandardScaler]:
    if scaler is None:
        scaler = StandardScaler()
        normalized = scaler.fit_transform(features.reshape(1, -1))
    else:
        normalized = scaler.transform(features.reshape(1, -1))
    return normalized.flatten(), scaler


def compute_request_fingerprint(
    method: str,
    path: str,
    headers: Dict[str, str],
    user_agent: str,
    body: Optional[str] = None,
    query_params: Optional[Dict[str, str]] = None,
) -> str:
    header_keys_sorted = sorted(headers.keys())
    canonical_headers = {k.lower(): headers[k] for k in header_keys_sorted}

    parts = [
        method.upper(),
        path.rstrip("/") or "/",
        str(sorted(query_params.items()) if query_params else []),
        str(sorted(canonical_headers.items())),
        user_agent.lower().strip(),
        (body or "").strip()[:256],
    ]

    raw = "|".join(parts)
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()
