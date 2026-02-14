from collections import defaultdict
from dataclasses import asdict
from typing import Callable, DefaultDict, Dict, List
import logging

from saga.models import Event


Handler = Callable[[Event], None]


class EventBus:
    def __init__(self, logger: logging.Logger) -> None:
        self.logger = logger
        self._handlers: DefaultDict[str, List[Handler]] = defaultdict(list)
        self.history: List[Event] = []

    def subscribe(self, event_name: str, handler: Handler) -> None:
        self._handlers[event_name].append(handler)
        self.logger.debug("handler_subscribed", extra={"event": event_name, "handler": handler.__name__})

    def publish(self, event: Event) -> None:
        self.history.append(event)
        handlers = self._handlers.get(event.name, [])

        self.logger.info(
            "event_published",
            extra={
                "event": event.name,
                "saga_id": event.saga_id,
                "payload": event.payload,
                "handler_count": len(handlers),
            },
        )

        for handler in handlers:
            self.logger.debug(
                "event_dispatched",
                extra={"event": event.name, "saga_id": event.saga_id, "handler": handler.__name__},
            )
            handler(event)

    def history_dict(self) -> List[Dict[str, object]]:
        return [asdict(event) for event in self.history]
