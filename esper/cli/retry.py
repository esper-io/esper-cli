"""
Retry utility for transient API failures.

Usage::

    from esper.cli.retry import with_retry
    from esperclient.rest import ApiException

    @with_retry(retries=3, backoff=1.0, exceptions=(ApiException,))
    def call_api():
        return client.get_all_devices(enterprise_id)

The decorator retries only on *transient* failures:

* The exception has a ``status`` attribute ≥ 500 (server-side error), or
* The exception is a ``ConnectionError`` / ``TimeoutError`` / ``OSError``.

Back-off is exponential: ``backoff * 2^attempt`` seconds between tries.
"""
from __future__ import annotations

import functools
import logging
import time
from typing import Callable, Tuple, Type

log = logging.getLogger("espercli")


def with_retry(
    retries: int = 3,
    backoff: float = 1.0,
    exceptions: Tuple[Type[Exception], ...] = (Exception,),
) -> Callable:
    """
    Decorator factory.  Returns a decorator that wraps *func* with retry
    logic.

    Parameters
    ----------
    retries:
        Maximum number of total attempts (default 3).
    backoff:
        Base wait time in seconds (doubles each attempt, default 1.0).
    exceptions:
        Only catch these exception types (default ``(Exception,)``).
    """
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            last_exc: Exception | None = None
            for attempt in range(retries):
                try:
                    return func(*args, **kwargs)
                except exceptions as exc:
                    last_exc = exc
                    # Only retry on genuinely transient failures
                    transient = (
                        isinstance(exc, (ConnectionError, TimeoutError, OSError))
                        or (hasattr(exc, "status") and isinstance(exc.status, int) and exc.status >= 500)
                    )
                    if not transient or attempt >= retries - 1:
                        raise
                    wait = backoff * (2 ** attempt)
                    log.debug(
                        "[retry] %s attempt %d/%d failed (%s); retrying in %.1fs…",
                        func.__name__, attempt + 1, retries, exc, wait,
                    )
                    time.sleep(wait)
            raise last_exc  # pragma: no cover

        return wrapper
    return decorator
