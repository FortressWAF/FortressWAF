import os
import math
import pickle
import logging
from datetime import datetime, timedelta, timezone
from typing import Dict, Optional, List, Tuple, Any
from collections import defaultdict, deque

import numpy as np
from sklearn.ensemble import GradientBoostingRegressor

from training.feature_engineering import compute_entropy

logger = logging.getLogger(__name__)

MODEL_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), "models", "persisted")
os.makedirs(MODEL_DIR, exist_ok=True)

RISK_MODEL_PATH = os.path.join(MODEL_DIR, "risk_model.pkl")

EARTH_RADIUS_KM = 6371.0


def haversine_distance(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
    dlat = math.radians(lat2 - lat1)
    dlon = math.radians(lon2 - lon1)
    a = math.sin(dlat / 2) ** 2 + math.cos(math.radians(lat1)) * math.cos(math.radians(lat2)) * math.sin(dlon / 2) ** 2
    c = 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a))
    return EARTH_RADIUS_KM * c


class SessionState:
    def __init__(self, session_id: str):
        self.session_id = session_id
        self.login_failures: int = 0
        self.successful_logins: int = 0
        self.last_login_time: Optional[datetime] = None
        self.request_count: int = 0
        self.first_seen: datetime = datetime.now(timezone.utc)
        self.last_seen: datetime = datetime.now(timezone.utc)
        self.last_latitude: Optional[float] = None
        self.last_longitude: Optional[float] = None
        self.last_ip: Optional[str] = None
        self.request_timestamps: deque = deque(maxlen=100)
        self.endpoints_accessed: set = set()
        self.payload_anomaly_scores: list = []
        self.failed_paths: set = set()
        self.auth_endpoints: set = set()
        self.sensitive_endpoints: set = set()
        self.method_counts: Dict[str, int] = defaultdict(int)
        self.status_code_counts: Dict[int, int] = defaultdict(int)

    def record_request(self, endpoint: str, method: str, status_code: int,
                       anomaly_score: float, is_auth: bool = False,
                       is_sensitive: bool = False, latitude: Optional[float] = None,
                       longitude: Optional[float] = None, ip_address: Optional[str] = None,
                       success: bool = True):
        now = datetime.now(timezone.utc)
        self.last_seen = now
        self.request_count += 1
        self.request_timestamps.append(now)
        self.endpoints_accessed.add(endpoint)
        self.method_counts[method] += 1
        self.status_code_counts[status_code] += 1
        self.payload_anomaly_scores.append(anomaly_score)

        if is_auth:
            self.auth_endpoints.add(endpoint)
            if success:
                self.successful_logins += 1
                self.last_login_time = now
            else:
                self.login_failures += 1

        if is_sensitive:
            self.sensitive_endpoints.add(endpoint)

        if latitude is not None and longitude is not None:
            if self.last_latitude is not None and self.last_longitude is not None:
                pass
            self.last_latitude = latitude
            self.last_longitude = longitude
            self.last_ip = ip_address

        if status_code in (401, 403, 404):
            self.failed_paths.add(endpoint)

    def get_session_duration_minutes(self) -> float:
        if not self.first_seen:
            return 0.0
        return (self.last_seen - self.first_seen).total_seconds() / 60.0

    def get_request_rate(self) -> float:
        duration = self.get_session_duration_minutes()
        if duration < 0.01:
            return float(self.request_count)
        return self.request_count / duration

    def get_unique_endpoint_ratio(self) -> float:
        if self.request_count == 0:
            return 0.0
        return len(self.endpoints_accessed) / self.request_count

    def get_failure_ratio(self) -> float:
        total_auth = self.login_failures + self.successful_logins
        if total_auth == 0:
            return 0.0
        return self.login_failures / total_auth

    def get_path_traversal_ratio(self) -> float:
        total = len(self.failed_paths) + len(self.endpoints_accessed)
        if total == 0:
            return 0.0
        return len(self.failed_paths) / total

    def get_anomaly_mean(self) -> float:
        if not self.payload_anomaly_scores:
            return 0.0
        return np.mean(self.payload_anomaly_scores[-50:])

    def get_anomaly_std(self) -> float:
        if len(self.payload_anomaly_scores) < 2:
            return 0.0
        return float(np.std(self.payload_anomaly_scores[-50:]))

    def get_method_diversity(self) -> float:
        total = sum(self.method_counts.values())
        if total == 0:
            return 0.0
        return len(self.method_counts) / 4.0  # normalize by common methods count

    def get_entropy_score(self) -> float:
        combined = "|".join(sorted(self.endpoints_accessed)) if self.endpoints_accessed else ""
        return compute_entropy(combined)


class RiskScorer:
    def __init__(self):
        self.model: Optional[GradientBoostingRegressor] = None
        self.sessions: Dict[str, SessionState] = {}
        self._geoip_cache: Dict[str, Tuple[float, float]] = {}
        self._load_model()

    def _compute_features(self, session: SessionState, current_anomaly: float = 0.0,
                          latitude: Optional[float] = None, longitude: Optional[float] = None) -> np.ndarray:
        features = []

        features.append(session.login_failures)
        features.append(session.successful_logins)
        features.append(session.get_failure_ratio())

        duration = session.get_session_duration_minutes()
        features.append(duration)
        features.append(session.get_request_rate())

        features.append(session.request_count)
        features.append(len(session.endpoints_accessed))
        features.append(session.get_unique_endpoint_ratio())

        features.append(session.get_path_traversal_ratio())
        features.append(session.get_anomaly_mean())
        features.append(session.get_anomaly_std())
        features.append(current_anomaly)

        features.append(session.get_method_diversity())
        features.append(session.get_entropy_score())

        features.append(len(session.auth_endpoints))
        features.append(len(session.sensitive_endpoints))

        features.append(session.status_code_counts.get(401, 0))
        features.append(session.status_code_counts.get(403, 0))
        features.append(session.status_code_counts.get(404, 0))
        features.append(session.status_code_counts.get(500, 0))

        time_since_last_login = 0.0
        if session.last_login_time is not None:
            time_since_last_login = (datetime.now(timezone.utc) - session.last_login_time).total_seconds() / 60.0
        features.append(time_since_last_login)

        recent_window = list(session.request_timestamps)[-20:] if len(session.request_timestamps) >= 20 else list(session.request_timestamps)
        if len(recent_window) >= 2:
            intervals = [(recent_window[i + 1] - recent_window[i]).total_seconds() for i in range(len(recent_window) - 1)]
            features.append(np.std(intervals) if intervals else 0.0)
            features.append(np.mean(intervals) if intervals else 0.0)
        else:
            features.append(0.0)
            features.append(0.0)

        features.append(session.last_seen.hour)
        features.append(session.last_seen.weekday())

        if latitude is not None and longitude is not None and session.last_latitude is not None and session.last_longitude is not None:
            distance = haversine_distance(session.last_latitude, session.last_longitude, latitude, longitude)
            time_diff = (datetime.now(timezone.utc) - session.last_seen).total_seconds() / 3600.0
            if time_diff > 0:
                travel_speed = distance / time_diff
                features.append(distance)
                features.append(min(travel_speed / 1000.0, 100.0))
            else:
                features.append(0.0)
                features.append(0.0)
        else:
            features.append(0.0)
            features.append(0.0)

        if len(features) < 30:
            features.extend([0.0] * (30 - len(features)))

        return np.array(features[:30], dtype=np.float64)

    def detect_impossible_travel(self, session: SessionState, latitude: float, longitude: float) -> Tuple[bool, float]:
        if session.last_latitude is None or session.last_longitude is None:
            return False, 0.0

        distance = haversine_distance(session.last_latitude, session.last_longitude, latitude, longitude)
        time_diff_hours = (datetime.now(timezone.utc) - session.last_seen).total_seconds() / 3600.0

        if time_diff_hours <= 0:
            return False, 0.0

        speed = distance / time_diff_hours
        if speed > 900 and distance > 100:
            return True, min(speed / 2000.0, 1.0)

        return False, 0.0

    def detect_ato(self, session: SessionState) -> Tuple[bool, float]:
        if session.login_failures >= 5 and session.get_failure_ratio() > 0.8:
            return True, min(0.5 + session.login_failures * 0.05, 1.0)

        if session.login_failures > 0 and session.get_request_rate() > 100:
            return True, min(0.6 + session.get_request_rate() * 0.001, 1.0)

        time_since_first = (datetime.now(timezone.utc) - session.first_seen).total_seconds()
        if time_since_first < 60 and session.login_failures >= 3:
            return True, 0.7

        return False, 0.0

    def score(self, session_id: str, current_anomaly: float = 0.0,
              latitude: Optional[float] = None, longitude: Optional[float] = None) -> Tuple[int, Dict[str, Any]]:
        if session_id not in self.sessions:
            self.sessions[session_id] = SessionState(session_id)

        session = self.sessions[session_id]

        travel_detected, travel_score = False, 0.0
        if latitude is not None and longitude is not None:
            travel_detected, travel_score = self.detect_impossible_travel(session, latitude, longitude)

        ato_detected, ato_score = self.detect_ato(session)

        features = self._compute_features(session, current_anomaly, latitude, longitude).reshape(1, -1)

        if self.model is None and features.shape[1] > 0:
            base_score = 0
        else:
            try:
                base_score = int(self.model.predict(features)[0])
            except Exception:
                base_score = 0

        if travel_detected:
            base_score = min(base_score + int(travel_score * 40), 100)

        if ato_detected:
            base_score = min(base_score + int(ato_score * 30), 100)

        if current_anomaly > 0.8:
            base_score = min(base_score + 20, 100)

        if session.login_failures > 10:
            base_score = min(base_score + 15, 100)

        if session.get_request_rate() > 200:
            base_score = min(base_score + 10, 100)

        details = {
            "impossible_travel": travel_detected,
            "travel_score": travel_score,
            "ato_detected": ato_detected,
            "ato_score": ato_score,
            "login_failures": session.login_failures,
            "request_rate": round(session.get_request_rate(), 2),
            "anomaly_mean": round(session.get_anomaly_mean(), 4),
        }

        return min(max(base_score, 0), 100), details

    def record_request(self, session_id: str, endpoint: str, method: str,
                       status_code: int, anomaly_score: float,
                       is_auth: bool = False, is_sensitive: bool = False,
                       success: bool = True, latitude: Optional[float] = None,
                       longitude: Optional[float] = None, ip_address: Optional[str] = None):
        if session_id not in self.sessions:
            self.sessions[session_id] = SessionState(session_id)

        self.sessions[session_id].record_request(
            endpoint=endpoint, method=method, status_code=status_code,
            anomaly_score=anomaly_score, is_auth=is_auth, is_sensitive=is_sensitive,
            latitude=latitude, longitude=longitude, ip_address=ip_address, success=success,
        )

    def cleanup_old_sessions(self, max_age_hours: int = 24):
        now = datetime.now(timezone.utc)
        cutoff = now - timedelta(hours=max_age_hours)
        expired = [sid for sid, sess in self.sessions.items() if sess.last_seen < cutoff]
        for sid in expired:
            del self.sessions[sid]
        logger.info(f"Cleaned up {len(expired)} expired sessions")

    def train(self, features_batch: np.ndarray, labels: np.ndarray):
        self.model = GradientBoostingRegressor(
            n_estimators=200,
            max_depth=5,
            learning_rate=0.1,
            min_samples_leaf=5,
            subsample=0.8,
            random_state=42,
        )
        self.model.fit(features_batch, labels)
        self._save_model()

    def _save_model(self):
        if self.model is not None:
            path = RISK_MODEL_PATH
            with open(path, "wb") as f:
                pickle.dump(self.model, f)
            logger.info(f"Risk model saved to {path}")

    def _load_model(self):
        path = RISK_MODEL_PATH
        if os.path.exists(path):
            try:
                with open(path, "rb") as f:
                    self.model = pickle.load(f)
                logger.info(f"Risk model loaded from {path}")
            except Exception as e:
                logger.error(f"Failed to load risk model: {e}")
                self.model = None

    @property
    def version(self) -> str:
        return f"gradient-boosting-risk-v2.0"

    @property
    def is_loaded(self) -> bool:
        return self.model is not None
