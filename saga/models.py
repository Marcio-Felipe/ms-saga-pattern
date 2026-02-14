from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, List


class SagaStatus(str, Enum):
    STARTED = "STARTED"
    IN_PROGRESS = "IN_PROGRESS"
    COMPLETED = "COMPLETED"
    FAILED = "FAILED"
    FAILED_COMPENSATED = "FAILED_COMPENSATED"


@dataclass
class Event:
    name: str
    saga_id: str
    payload: Dict[str, Any] = field(default_factory=dict)


@dataclass
class SagaResult:
    saga_id: str
    status: SagaStatus
    steps: List[str] = field(default_factory=list)
    compensations: List[str] = field(default_factory=list)
    errors: List[str] = field(default_factory=list)
