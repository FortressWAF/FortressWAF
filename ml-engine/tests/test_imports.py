import sys

errors = 0

def test_import(name, import_path):
    global errors
    try:
        exec(f"from {import_path} import *")
        print(f"  ✓ {name}")
    except Exception as e:
        print(f"  ✗ {name}: {e}")
        errors += 1

test_import("FastAPI app", "api.app")
test_import("AnomalyDetector", "models.anomaly")
test_import("AttackClassifier", "models.classifier")
test_import("BotDetector", "models.bot_detector")
test_import("RiskScorer", "models.risk_scorer")

sys.exit(errors)
