from typing import Dict, Optional
from pydantic import BaseModel, Field


class InspectRequest(BaseModel):
    method: str
    path: str
    headers: Dict[str, str]
    body: Optional[str] = None
    query_params: Dict[str, str] = {}
    source_ip: str
    user_agent: str
    content_type: Optional[str] = None


class InspectResponse(BaseModel):
    anomaly_score: float
    is_anomaly: bool
    attack_type: Optional[str] = None
    attack_confidence: Optional[float] = None
    bot_score: float
    risk_score: int
    fingerprint: str
    model_version: str


class ClassifyRequest(BaseModel):
    payload: str
    context: Optional[Dict[str, str]] = None


class ClassifyResponse(BaseModel):
    attack_type: str
    confidence: float
    model_version: str


class FingerprintRequest(BaseModel):
    method: str
    path: str
    headers: Dict[str, str] = {}
    user_agent: str


class FingerprintResponse(BaseModel):
    fingerprint: str
    hash_algorithm: str


class BotScoreRequest(BaseModel):
    user_agent: str
    headers: Dict[str, str] = {}
    request_timing_ms: int
    js_challenge_result: Optional[bool] = None
    accept_language: str


class BotScoreResponse(BaseModel):
    bot_score: float
    is_bot: bool
    bot_type: Optional[str] = None


class ModelStatus(BaseModel):
    model_version: str
    last_trained: str
    anomaly_model_loaded: bool
    classifier_model_loaded: bool
    bot_model_loaded: bool
    risk_model_loaded: bool
    total_samples_trained: int


class RetrainConfig(BaseModel):
    learning_rate: float = Field(default=0.01, ge=0.0, le=1.0)
    n_estimators: int = Field(default=100, ge=10, le=1000)
    batch_size: int = Field(default=32, ge=1, le=1024)
    force_full_retrain: bool = False
