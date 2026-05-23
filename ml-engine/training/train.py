#!/usr/bin/env python3
"""
FortressWAF Training Script
Usage: python -m training.train --data-dir /path/to/corpus
"""

import os
import sys
import json
import argparse
import logging
from typing import List, Tuple, Optional
from datetime import datetime

import numpy as np
import pandas as pd
from sklearn.model_selection import cross_val_score, StratifiedKFold, train_test_split
from sklearn.metrics import classification_report, accuracy_score, f1_score, mean_squared_error
from sklearn.ensemble import RandomForestClassifier, GradientBoostingRegressor
from sklearn.linear_model import LogisticRegression
from sklearn.feature_extraction.text import TfidfVectorizer

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from models.anomaly import AnomalyDetector, ANOMALY_MODEL_PATH
from models.classifier import AttackClassifier, ATTACK_CLASSES, CLASSIFIER_MODEL_PATH
from models.bot_detector import BotDetector, BOT_MODEL_PATH
from models.risk_scorer import RiskScorer, RISK_MODEL_PATH
from training.feature_engineering import extract_http_features, build_tfidf_vectorizer

logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)


def load_attack_corpus(data_dir: str) -> Tuple[List[str], List[str]]:
    texts = []
    labels = []

    if not os.path.exists(data_dir):
        logger.warning(f"Data directory {data_dir} does not exist. Using synthetic data.")
        return _generate_synthetic_data()

    attack_dirs = [d for d in os.listdir(data_dir)
                   if os.path.isdir(os.path.join(data_dir, d))]

    if not attack_dirs:
        logger.warning(f"No attack data found in {data_dir}. Using synthetic data.")
        return _generate_synthetic_data()

    for attack_type in attack_dirs:
        attack_path = os.path.join(data_dir, attack_type)
        for fname in os.listdir(attack_path):
            if fname.endswith((".txt", ".json", ".csv", ".log")):
                fpath = os.path.join(attack_path, fname)
                try:
                    if fname.endswith(".json"):
                        with open(fpath) as f:
                            data = json.load(f)
                            if isinstance(data, list):
                                for item in data:
                                    payload = item.get("payload", item.get("request", json.dumps(item)))
                                    texts.append(payload)
                                    labels.append(attack_type)
                            elif isinstance(data, dict):
                                payload = data.get("payload", data.get("request", ""))
                                if payload:
                                    texts.append(payload)
                                    labels.append(attack_type)
                    else:
                        with open(fpath, errors="ignore") as f:
                            for line in f:
                                line = line.strip()
                                if line:
                                    texts.append(line)
                                    labels.append(attack_type)
                except Exception as e:
                    logger.error(f"Error reading {fpath}: {e}")

    if not texts:
        logger.warning("No data loaded from corpus. Using synthetic data.")
        return _generate_synthetic_data()

    logger.info(f"Loaded {len(texts)} samples from attack corpus")
    return texts, labels


def _generate_synthetic_data() -> Tuple[List[str], List[str]]:
    np.random.seed(42)
    texts = []
    labels = []

    attack_patterns = {
        "sql-injection": [
            "' OR '1'='1", "'; DROP TABLE users; --", "1 UNION SELECT * FROM users",
            "admin'--", "1' AND 1=1", "' OR '1'='1' --", "1' ORDER BY 1--",
            "' UNION SELECT username,password FROM users--",
        ],
        "xss": [
            "<script>alert('xss')</script>", "<img src=x onerror=alert(1)>",
            "<svg/onload=alert(1)>", "javascript:alert(document.cookie)",
            "<body onload=alert(1)>", "<input onfocus=alert(1)>",
        ],
        "rce": [
            "; cat /etc/passwd", "| ls -la", "$(cat /etc/passwd)", "`id`",
            "&& whoami", "| nc -e /bin/sh", "; bash -c 'cat /etc/shadow'",
        ],
        "path-traversal": [
            "../../etc/passwd", "..\\..\\windows\\system32\\config",
            "%2e%2e%2fetc%2fpasswd", "....//....//etc/passwd",
            "../../../../etc/shadow", "..\\Windows\\System32\\cmd.exe",
        ],
        "ssrf": [
            "http://169.254.169.254/latest/meta-data/",
            "http://127.0.0.1:8080/admin", "file:///etc/passwd",
            "gopher://internal:6379", "dict://localhost:6379/info",
        ],
        "lfi": [
            "/etc/passwd", "/proc/self/environ", "/var/log/apache/access.log",
            "../../../../etc/issue", "/etc/shadow",
        ],
        "command-injection": [
            "127.0.0.1; ls -la", "127.0.0.1|cat /etc/passwd",
            "127.0.0.1$(id)", "127.0.0.1`id`", "8.8.8.8 && whoami",
        ],
        "ssti": [
            "{{7*7}}", "${7*7}", "{{config}}", "${user.name}",
            "{% if 1==1 %}yes{% endif %}", "{{ ''.__class__.__mro__ }}",
        ],
        "no-attack": [
            "Hello world", "How are you?", "Please login to continue",
            "Search results for query", "Welcome to our website",
            "This is a test", "Normal request body",
            "Update your profile information", "Submit feedback form",
            "View product details for item #12345",
        ],
    }

    for attack_type, patterns in attack_patterns.items():
        for i in range(50):
            for p in patterns:
                texts.append(p + str(i))
                labels.append(attack_type)

    combined = list(zip(texts, labels))
    np.random.shuffle(combined)
    texts, labels = zip(*combined) if combined else ([], [])

    logger.info(f"Generated {len(texts)} synthetic training samples")
    return list(texts), list(labels)


def train_anomaly_model(data: List[dict], output_dir: str):
    logger.info("Training anomaly detection model...")

    if not data:
        logger.warning("No data for anomaly training, using synthetic data")
        np.random.seed(42)
        data = []
        for _ in range(500):
            data.append({
                "method": np.random.choice(["GET", "POST", "PUT", "DELETE"]),
                "path": np.random.choice(["/", "/api/users", "/login", "/admin", "/search"]),
                "headers": {},
                "body": None if np.random.random() > 0.5 else "test",
                "query_params": {} if np.random.random() > 0.3 else {"q": "test"},
            })

    detector = AnomalyDetector()
    features_list = []
    for req in data:
        feats = detector.extract_features(req)
        features_list.append(feats)

    features_batch = np.array(features_list)
    if features_batch.shape[1] != 38:
        features_batch = np.pad(features_batch, ((0, 0), (0, max(0, 38 - features_batch.shape[1]))), mode="constant")[:, :38]

    detector.train(features_batch)

    logger.info(f"Anomaly model trained on {len(features_batch)} samples")
    cv_scores = cross_val_score(detector.model, features_batch,
                                np.ones(len(features_batch)), cv=3, scoring="neg_log_loss")
    logger.info(f"Cross-validation scores: {cv_scores}")

    return detector


def train_classifier(texts: List[str], labels: List[str], output_dir: str):
    logger.info("Training attack classifier...")

    classifier = AttackClassifier()

    X_train, X_test, y_train, y_test = train_test_split(
        texts, labels, test_size=0.2, random_state=42, stratify=labels
    )

    classifier.train_fallback(X_train, y_train)

    y_pred = []
    for t in X_test:
        pred_type, _ = classifier.predict(t)
        y_pred.append(pred_type)

    accuracy = accuracy_score(y_test, y_pred)
    f1 = f1_score(y_test, y_pred, average="weighted")
    logger.info(f"Classifier accuracy: {accuracy:.4f}, F1: {f1:.4f}")

    try:
        report = classification_report(y_test, y_pred, zero_division=0)
        logger.info(f"Classification report:\n{report}")
    except Exception as e:
        logger.error(f"Classification report error: {e}")

    classifier.export_bert()

    return classifier


def train_bot_model(data_dir: str, output_dir: str):
    logger.info("Training bot detection model...")

    np.random.seed(42)
    n_samples = 1000
    features_list = []
    labels_list = []

    detector = BotDetector()

    for i in range(n_samples):
        is_bot = np.random.random() < 0.3
        ua = "Mozilla/5.0 Bot/1.0" if is_bot else "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
        headers = {"user-agent": ua, "accept": "*/*" if is_bot else "text/html", "accept-language": "en-US,en;q=0.9"}
        timing = np.random.randint(1, 100) if is_bot else np.random.randint(100, 5000)
        lang = "en-US,en;q=0.9"

        feats = detector._extract_features(ua, headers, timing, lang, None)
        if len(feats) >= 30:
            features_list.append(feats[:30])
            labels_list.append(1 if is_bot else 0)

    features_batch = np.array(features_list)
    labels_batch = np.array(labels_list)

    if features_batch.shape[0] > 0 and features_batch.shape[1] > 0:
        X_train, X_test, y_train, y_test = train_test_split(
            features_batch, labels_batch, test_size=0.2, random_state=42
        )

        detector.train(X_train, y_train)

        y_pred = detector.model.predict(X_test)
        accuracy = accuracy_score(y_test, y_pred)
        logger.info(f"Bot model accuracy: {accuracy:.4f}")

        cv_scores = cross_val_score(detector.model, features_batch, labels_batch, cv=3)
        logger.info(f"Bot model CV scores: {cv_scores}")

    return detector


def train_risk_model(data_dir: str, output_dir: str):
    logger.info("Training risk scoring model...")

    np.random.seed(42)
    n_samples = 1000
    features_list = []
    scores_list = []

    for i in range(n_samples):
        login_failures = np.random.poisson(0.5)
        success_logins = np.random.poisson(2)
        request_count = np.random.poisson(20)
        anomaly_mean = np.random.beta(0.5, 5)
        unique_endpoints = min(request_count, np.random.poisson(5))

        score = min(
            login_failures * 8
            + (1 - success_logins / max(success_logins + login_failures, 1)) * 20
            + anomaly_mean * 30
            + (1 - unique_endpoints / max(request_count, 1)) * 15
            + max(0, np.random.randn() * 5),
            100,
        )

        features = np.array([
            login_failures, success_logins,
            login_failures / max(login_failures + success_logins, 1),
            10.0, request_count / 10.0, request_count, unique_endpoints,
            unique_endpoints / max(request_count, 1),
            0.0, anomaly_mean, 0.0, anomaly_mean,
            0.5, 0.3,
            0, 0, 0, 0, 0, 0, 0.0, 0.0, 0.0, 12.0, 3.0, 0.0, 0.0,
        ])

        if len(features) < 30:
            features = np.pad(features, (0, 30 - len(features)), mode="constant")
        features_list.append(features[:30])
        scores_list.append(score)

    features_batch = np.array(features_list)
    scores_batch = np.array(scores_list)

    if features_batch.shape[0] > 10:
        X_train, X_test, y_train, y_test = train_test_split(
            features_batch, scores_batch, test_size=0.2, random_state=42
        )

        risk_scorer = RiskScorer()
        risk_scorer.train(X_train, y_train)

        y_pred = risk_scorer.model.predict(X_test)
        mse = mean_squared_error(y_test, y_pred)
        logger.info(f"Risk model MSE: {mse:.4f}")

        return risk_scorer

    return RiskScorer()


def main():
    parser = argparse.ArgumentParser(description="FortressWAF ML Engine Training")
    parser.add_argument("--data-dir", type=str, default=None,
                        help="Path to attack corpus directory")
    parser.add_argument("--output-dir", type=str, default=None,
                        help="Path to output directory for models")
    args = parser.parse_args()

    data_dir = args.data_dir
    output_dir = args.output_dir or os.path.join(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "models", "persisted"
    )
    os.makedirs(output_dir, exist_ok=True)

    logger.info(f"Data directory: {data_dir or 'None (using synthetic data)'}")
    logger.info(f"Output directory: {output_dir}")

    texts, labels = [], []
    if data_dir:
        texts, labels = load_attack_corpus(data_dir)

    anomaly_data = []
    if data_dir:
        anomaly_path = os.path.join(data_dir, "requests.json")
        if os.path.exists(anomaly_path):
            try:
                with open(anomaly_path) as f:
                    anomaly_data = json.load(f)
            except Exception:
                pass

    train_anomaly_model(anomaly_data, output_dir)

    if texts and labels:
        train_classifier(texts, labels, output_dir)
    else:
        logger.info("No classification corpus found, using synthetic data")
        synth_texts, synth_labels = _generate_synthetic_data()
        train_classifier(synth_texts, synth_labels, output_dir)

    train_bot_model(data_dir, output_dir)
    train_risk_model(data_dir, output_dir)

    logger.info("=" * 60)
    logger.info("Training complete! All models saved.")
    logger.info(f"Model directory: {output_dir}")
    logger.info("=" * 60)


if __name__ == "__main__":
    main()
