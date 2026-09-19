"""Read-only CLI comparisons. Persist shapes and differences, never raw payloads."""

import concurrent.futures
import datetime
import hashlib
import json
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
OUTPUT = pathlib.Path(__file__).with_name("comparisons.jsonl")
ENTERPRISE = "f44373cb-1800-43c6-aab3-c81f8b1f435c"
DEVICE = "fc4eb40e-6310-422c-b764-44070281c508"
THIRD_DEVICE = "7ddfa60d-036e-4bba-8606-0933ae65bb4c"
APP = "8c192b39-ad44-4a4f-a48d-eb49756ffd58"
VERSION = "eb474b2f-ad85-48d3-9c63-457487fc9c76"
REQUEST = "50b2ed1e-f439-48e9-9f60-c3c25778dc89"
PAGE = ["--limit", "3", "--offset", "0"]
E = ["--enterprise", ENTERPRISE]
D = E + ["--device", DEVICE]
AV = E + ["--application", APP, "--version", VERSION] + PAGE


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def scrub(text):
    text = re.sub(r"https?://[^\s\"<>]+", "<url>", text)
    text = re.sub(r"[\w.+-]+@[\w.-]+", "<email>", text)
    text = re.sub(r"(?i)(token|api[_-]?key|authorization)([\s\"':=]+)[^\s,}]+",
                  r"\1\2<redacted>", text)
    return text[:1500]


def shape(value, depth=0):
    if depth >= 5:
        return type(value).__name__
    if isinstance(value, dict):
        return {key: shape(item, depth + 1) for key, item in sorted(value.items())}
    if isinstance(value, list):
        return [json.loads(item) for item in sorted({
            json.dumps(shape(item, depth + 1), sort_keys=True) for item in value[:3]
        })]
    return type(value).__name__


def content(value):
    return value.get("content", value) if isinstance(value, dict) else value


def rows(value):
    value = content(value)
    if isinstance(value, dict) and "results" in value and value["results"] is None:
        return []
    if isinstance(value, dict) and isinstance(value.get("results"), list):
        return value["results"]
    return value if isinstance(value, list) else [value] if isinstance(value, dict) else []


def diff(left, right, path="$"):
    if type(left) is not type(right):
        return [path + " (type)"]
    if isinstance(left, dict):
        result = []
        for key in sorted(left.keys() | right.keys()):
            child = path + "." + key
            if key not in left or key not in right:
                result.append(child + " (missing)")
            else:
                result.extend(diff(left[key], right[key], child))
        return result
    if isinstance(left, list):
        result = [path + " (length)"] if len(left) != len(right) else []
        for a, b in zip(left[:3], right[:3]):
            result.extend(diff(a, b, path + "[]"))
        return sorted(set(result))
    return [] if left == right else [path]


def run(binary, operations, command, arguments):
    parts = command.split()
    supplied = {arg for arg in arguments if arg.startswith("--")}
    candidates = [o for o in operations if o["Command"] == parts and not o.get("AliasOf")]
    all_scopes = {"--" + p["ScopeName"] for o in candidates
                  for p in o.get("Parameters") or [] if p.get("Scope")}
    candidates = [o for o in candidates if {
        "--" + p["ScopeName"] for p in o.get("Parameters") or [] if p.get("Scope")
    } == supplied & all_scopes]
    if len(candidates) != 1 or candidates[0]["Method"] != "GET":
        raise ValueError("Not one unambiguous GET: " + command)
    operation = candidates[0]
    argv = [str(binary), *parts, *arguments, "--json", "--environment", "develop"]
    start = now()
    try:
        process = subprocess.run(argv, cwd=ROOT, text=True, capture_output=True, timeout=70)
        output, error, code = process.stdout, process.stderr, process.returncode
    except subprocess.TimeoutExpired:
        output = ""
        code = None
        error = "Process timeout after 70 seconds; no exit code available"
    try:
        value = json.loads(output)
    except json.JSONDecodeError:
        value = None
    payload = content(value)
    record = {
        "kind": "request", "command": command, "argv": argv,
        "method": operation["Method"], "path": operation["Path"],
        "started_at": start, "finished_at": now(), "exit_code": code,
        "stderr": scrub(error), "json_received": value is not None,
        "shape": shape(value), "returned_rows": len(rows(value)),
        "count": payload.get("count") if isinstance(payload, dict) else None,
        "next_present": bool(payload.get("next")) if isinstance(payload, dict) else None,
    }
    return record, value


def main():
    if OUTPUT.exists():
        raise SystemExit("Evidence already exists; review it instead of replaying the batch.")
    binary = pathlib.Path(sys.argv[1]).resolve()
    source = (ROOT / "internal/cmd/generated/zz_generated_commands.go").read_text()
    operations = json.loads(json.loads(source.split("[]byte(", 1)[1].rsplit(")", 1)[0]))
    cases = [
        ("application collection", ("api legacy application list", E + PAGE), ("application list", E + PAGE)),
        ("application item", ("api legacy application get", [ENTERPRISE, APP]), ("application get", [ENTERPRISE, APP])),
        ("application versions", ("api legacy version list", E + ["--application", APP] + PAGE), ("api v1 version list", E + ["--application", APP] + PAGE)),
        ("application version item", ("api legacy app-version get", [ENTERPRISE, APP, VERSION]), ("app-version get", [ENTERPRISE, APP, VERSION])),
        ("version installations", ("api legacy installdevice list", AV), ("installdevice list", AV)),
        ("device item", ("api legacy device get", [ENTERPRISE, DEVICE]), ("device get", [DEVICE])),
        ("device request item", ("device-request get", [DEVICE]), ("device get", [DEVICE])),
        ("device collection", ("api legacy device list", E + ["--search", "DEVE-LOP-X3FO9"] + PAGE), ("device list", ["--search", "DEVE-LOP-X3FO9"] + PAGE)),
        ("device apps", ("api legacy app list", D + PAGE), ("device-app list", ["--device", DEVICE] + PAGE)),
        ("app noun collision", ("api legacy app list", D + PAGE), ("app list", PAGE)),
        ("application catalogs", ("application list", E + PAGE), ("tenant-app list", PAGE)),
        ("event feeds", ("device-eventfeed list", D + PAGE), ("event-feed list", D + PAGE)),
        ("status metrics", ("api v1 status-metric list", E), ("status-metric list", E)),
        ("geofences", ("api v0 geofence list", E + PAGE), ("geofence list", PAGE)),
        ("command collections", ("api v0 command list", E + ["--devices", THIRD_DEVICE, "--command", "BEEP_DEVICE"]), ("command-request list", ["--devices", THIRD_DEVICE, "--command", "BEEP_DEVICE"] + PAGE)),
        ("command status", ("status get", [ENTERPRISE, REQUEST]), ("command-request-status list", ["--request", REQUEST] + PAGE)),
        ("event versus command status", ("api legacy status get", [ENTERPRISE, DEVICE, "--latest-event", "1"]), ("status get", [ENTERPRISE, REQUEST])),
        ("user identities", ("user get", ["21689"]), ("authn-user list", ["--legacy-user-id", "21689"])),
    ]
    cache = {}
    with OUTPUT.open("x") as stream:
        def save(record):
            stream.write(json.dumps(record, sort_keys=True) + "\n")
            stream.flush()
        save({"kind": "provenance", "binary": str(binary),
              "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
              "started_at": now(), "agent": "Astra", "environment": "develop"})
        for label, left, right in cases:
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                futures = {tuple([cmd, *args]): pool.submit(run, binary, operations, cmd, args)
                           for cmd, args in (left, right) if tuple([cmd, *args]) not in cache}
                for key, future in futures.items():
                    cache[key] = future.result()
                    save(cache[key][0])
            a, av = cache[tuple([left[0], *left[1]])]
            b, bv = cache[tuple([right[0], *right[1]])]
            comparable = a["exit_code"] == b["exit_code"] == 0 and av is not None and bv is not None
            result = {"kind": "comparison", "pair": label,
                      "left": a["argv"], "right": b["argv"], "both_succeeded": comparable,
                      "exact_equal": av == bv if comparable else None,
                      "different_fields": diff(av, bv) if comparable else None,
                      "note": "Bounded samples; equality does not prove write or full-filter compatibility."}
            save(result)
            print(json.dumps({**result, "left": left[0], "right": right[0]}), flush=True)


if __name__ == "__main__":
    main()
