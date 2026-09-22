# Thin helpers for direct REST calls not covered by the esperclient SDK.
import requests


def api_get_all(environment: str, api_key: str, path: str, page_size: int = 100, **params) -> list:
    """Fetch every page from a paginated REST endpoint and return all results.

    The endpoint must return a JSON object with ``results`` (list) and
    ``count`` (int) keys — the standard Esper pagination envelope.
    """
    url = f"https://{environment}-api.esper.cloud/api{path}"
    headers = {"Authorization": f"Bearer {api_key}"}
    all_items: list = []
    offset = 0
    while True:
        resp = requests.get(
            url,
            headers=headers,
            params={**params, "limit": page_size, "offset": offset},
            timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()
        page = data.get("results", data.get("content", []))
        all_items.extend(page)
        total = data.get("count", 0)
        if not page or len(all_items) >= total:
            break
        offset += page_size
    return all_items


def api_get(environment: str, api_key: str, path: str, **params) -> dict:
    url = f"https://{environment}-api.esper.cloud/api{path}"
    resp = requests.get(url, headers={"Authorization": f"Bearer {api_key}"}, params=params, timeout=10)
    resp.raise_for_status()
    return resp.json()


def api_post(environment: str, api_key: str, path: str, body: dict) -> dict:
    url = f"https://{environment}-api.esper.cloud/api{path}"
    resp = requests.post(
        url,
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
        json=body,
        timeout=10,
    )
    resp.raise_for_status()
    return resp.json()


def api_delete(environment: str, api_key: str, path: str) -> int:
    url = f"https://{environment}-api.esper.cloud/api{path}"
    resp = requests.delete(url, headers={"Authorization": f"Bearer {api_key}"}, timeout=10)
    return resp.status_code
