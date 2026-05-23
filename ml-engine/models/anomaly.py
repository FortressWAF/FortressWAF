import os
import math
import pickle
import logging
from typing import Dict, List, Optional, Tuple, Any
from collections import Counter

import numpy as np
from sklearn.ensemble import IsolationForest

from training.feature_engineering import extract_http_features, compute_entropy

logger = logging.getLogger(__name__)

MODEL_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), "models", "persisted")
os.makedirs(MODEL_DIR, exist_ok=True)

ANOMALY_MODEL_PATH = os.path.join(MODEL_DIR, "anomaly_model.pkl")
ANOMALY_SCALER_PATH = os.path.join(MODEL_DIR, "anomaly_scaler.pkl")


class AnomalyDetector:
    def __init__(self, contamination: float = 0.1, random_state: int = 42):
        self.contamination = contamination
        self.random_state = random_state
        self.model: Optional[IsolationForest] = None
        self.trained_samples: int = 0
        self._baseline_mean: Optional[np.ndarray] = None
        self._baseline_std: Optional[np.ndarray] = None
        self._feature_dim: int = 38
        self.load_model()

    def extract_features(self, request_data: Dict[str, Any]) -> np.ndarray:
        features = extract_http_features(
            method=request_data.get("method", "GET"),
            path=request_data.get("path", "/"),
            headers=request_data.get("headers", {}),
            body=request_data.get("body"),
            query_params=request_data.get("query_params"),
        )
        if len(features) != self._feature_dim:
            padded = np.zeros(self._feature_dim)
            padded[:min(len(features), self._feature_dim)] = features[:self._feature_dim]
            features = padded
        return features

    def _compute_baseline(self, features_batch: np.ndarray):
        if features_batch.shape[0] < 2:
            return
        self._baseline_mean = np.mean(features_batch, axis=0)
        self._baseline_std = np.std(features_batch, axis=0)
        self._baseline_std = np.where(self._baseline_std == 0, 1e-6, self._baseline_std)

    def _three_sigma_anomaly(self, features: np.ndarray) -> Tuple[float, List[int]]:
        if self._baseline_mean is None or self._baseline_std is None:
            return 0.0, []
        z_scores = np.abs((features - self._baseline_mean) / self._baseline_std)
        anomalous_indices = np.where(z_scores > 3.0)[0].tolist()
        z_mean = float(np.mean(z_scores))
        return min(z_mean / 5.0, 1.0), anomalous_indices

    def partial_fit(self, features_batch: np.ndarray):
        if self.model is None:
            self.model = IsolationForest(
                contamination=self.contamination,
                random_state=self.random_state,
                warm_start=True,
                n_estimators=50,
            )
        if features_batch.shape[0] == 0:
            return
        combined = np.vstack([self.model.offset_ if hasattr(self.model, "offset_") and self.trained_samples > 0
                              else np.zeros((0, features_batch.shape[1])),
                              features_batch]) if self.trained_samples > 0 else features_batch

        n_samples = min(features_batch.shape[0] * 5, 500)
        n_estimators = min(max(10, int(self.trained_samples / 100) * 10 + 50), 200)

        self.model = IsolationForest(
            contamination=self.contamination,
            random_state=self.random_state,
            warm_start=True,
            n_estimators=n_estimators,
            max_samples=min(n_samples, features_batch.shape[0]),
        )
        self.model.fit(combined)
        self.trained_samples += features_batch.shape[0]
        self._compute_baseline(combined)
        self.save_model()

    def train(self, features_batch: np.ndarray):
        self.model = IsolationForest(
            contamination=self.contamination,
            random_state=self.random_state,
            n_estimators=100,
            max_samples="auto",
            bootstrap=False,
            n_jobs=-1,
        )
        self.model.fit(features_batch)
        self.trained_samples = features_batch.shape[0]
        self._compute_baseline(features_batch)
        self.save_model()

    def score(self, request_data: Dict[str, Any]) -> float:
        features = self.extract_features(request_data).reshape(1, -1)

        if self.model is None:
            self.model = IsolationForest(
                contamination=self.contamination,
                random_state=self.random_state,
                n_estimators=100,
            )
            dummy = np.random.randn(10, features.shape[1])
            self.model.fit(dummy)
            self.trained_samples = 10

        raw_score = self.model.decision_function(features)[0]
        normalized_score = 1.0 - (raw_score + 0.5)

        z_score, _ = self._three_sigma_anomaly(features.flatten())
        combined = max(normalized_score, z_score * 0.3)
        return max(0.0, min(1.0, combined))

    def is_anomaly(self, request_data: Dict[str, Any], threshold: float = 0.7) -> Tuple[bool, float]:
        score = self.score(request_data)
        return score >= threshold, score

    def save_model(self):
        if self.model is not None:
            path = ANOMALY_MODEL_PATH
            with open(path, "wb") as f:
                pickle.dump({
                    "model": self.model,
                    "trained_samples": self.trained_samples,
                    "baseline_mean": self._baseline_mean,
                    "baseline_std": self._baseline_std,
                    "feature_dim": self._feature_dim,
                }, f)
            logger.info(f"Anomaly model saved to {path}")

    def load_model(self):
        path = ANOMALY_MODEL_PATH
        if os.path.exists(path):
            try:
                with open(path, "rb") as f:
                    data = pickle.load(f)
                self.model = data["model"]
                self.trained_samples = data["trained_samples"]
                self._baseline_mean = data.get("baseline_mean")
                self._baseline_std = data.get("baseline_std")
                self._feature_dim = data.get("feature_dim", 38)
                logger.info(f"Anomaly model loaded from {path} ({self.trained_samples} samples)")
            except Exception as e:
                logger.error(f"Failed to load anomaly model: {e}")
                self.model = None

    @property
    def version(self) -> str:
        return f"isolation-forest-v1.{self.trained_samples // 1000 if self.trained_samples else 0}"

    @property
    def is_loaded(self) -> bool:
        return self.model is not None
