"""
HTML dashboard generator for `espercli device report`.

Produces a fully self-contained HTML file (no external assets except
Chart.js from a CDN) that can be opened in any browser.
"""
from __future__ import annotations

import html as _html
import json

# ---------------------------------------------------------------------------
# Colour helpers
# ---------------------------------------------------------------------------

def _battery_color(level) -> str:
    if level is None:
        return "#94a3b8"
    if level > 60:
        return "#4ade80"
    if level > 20:
        return "#fbbf24"
    return "#f87171"


def _battery_bg(level) -> str:
    if level is None:
        return "#1e293b"
    if level > 60:
        return "rgba(74,222,128,0.08)"
    if level > 20:
        return "rgba(251,191,36,0.08)"
    return "rgba(248,113,113,0.08)"


def _state_badge(state: str) -> str:
    styles = {
        "ACTIVE":   ("badge-green",  "✓ ACTIVE"),
        "INACTIVE": ("badge-red",    "✗ INACTIVE"),
        "DISABLED": ("badge-muted",  "— DISABLED"),
    }
    cls, label = styles.get(state.upper(), ("badge-muted", state))
    return f'<span class="{cls}">{label}</span>'


def _signal_dots(level) -> str:
    """Render 5 signal-strength dots."""
    if level is None:
        return '<span class="muted">—</span>'
    filled = int(level)
    dots = ""
    for i in range(5):
        cls = "dot-filled" if i < filled else "dot-empty"
        dots += f'<span class="{cls}">●</span>'
    return dots


def _mb_display(mb) -> str:
    if mb is None:
        return "—"
    gb = mb / 1024
    if gb >= 1:
        return f"{gb:.1f} GB"
    return f"{mb:,} MB"


def _cmd_state_class(state: str) -> str:
    s = (state or "").upper()
    if "SUCCESS" in s:
        return "cmd-success"
    if "FAIL" in s or "ERROR" in s:
        return "cmd-fail"
    return "cmd-neutral"


# ---------------------------------------------------------------------------
# Main generator
# ---------------------------------------------------------------------------

def generate_html(d: dict) -> str:
    """
    *d* keys:
      name, hardware_name, device_id, state, api_level, is_gms,
      generated_at, status (dict), battery_telemetry (list of {x,y}),
      temp_telemetry (list of {x,y}), commands (list of dicts),
      installs_count (int)
    """
    name        = _html.escape(str(d.get("name", "Unknown Device")))
    hw_name     = _html.escape(str(d.get("hardware_name", "—")))
    dev_id      = _html.escape(str(d.get("device_id", "—")))
    state_val   = _html.escape(str(d.get("state", "UNKNOWN")))
    api_level   = d.get("api_level", "—")
    is_gms      = "Yes" if d.get("is_gms") else "No"
    generated   = d.get("generated_at", "")

    status      = d.get("status") or {}
    bat_level   = status.get("battery_level")
    bat_temp    = status.get("battery_temperature")
    storage_mb  = status.get("memory_storage")
    ram_mb      = status.get("memory_ram")
    link_speed  = status.get("link_speed")
    sig_str     = status.get("signal_strength")

    bat_pct_display = f"{bat_level}%" if bat_level is not None else "—"
    bat_pct_val     = bat_level if bat_level is not None else 0
    bat_temp_display = f"{bat_temp}°C" if bat_temp is not None else "—"
    bat_color       = _battery_color(bat_level)
    bat_bg          = _battery_bg(bat_level)
    link_display    = f"{link_speed} Mbps" if link_speed is not None else "—"

    installs_count  = d.get("installs_count", 0)

    battery_data    = d.get("battery_telemetry") or []
    temp_data       = d.get("temp_telemetry") or []
    commands        = d.get("commands") or []

    # Build chart data JSON for injection
    bat_labels  = json.dumps([p["x"] for p in battery_data])
    bat_values  = json.dumps([p["y"] for p in battery_data])
    temp_labels = json.dumps([p["x"] for p in temp_data])
    temp_values = json.dumps([p["y"] for p in temp_data])

    # Commands table rows
    cmd_rows = ""
    for cmd in commands:
        cmd_date    = _html.escape(str(cmd.get("date", "")))
        cmd_command = _html.escape(str(cmd.get("command", "")))
        cmd_by      = _html.escape(str(cmd.get("issued_by", "")))
        cmd_state   = _html.escape(str(cmd.get("state", "")))
        state_cls   = _cmd_state_class(cmd_state)
        cmd_rows += (
            f'<tr>'
            f'<td class="td-date muted">{cmd_date}</td>'
            f'<td class="td-cmd"><code>{cmd_command}</code></td>'
            f'<td class="td-by muted">{cmd_by}</td>'
            f'<td><span class="cmd-badge {state_cls}">{cmd_state}</span></td>'
            f'</tr>'
        )
    if not cmd_rows:
        cmd_rows = '<tr><td colspan="4" class="muted td-empty">No recent commands</td></tr>'

    # No-data placeholders for charts
    bat_chart_empty = "" if battery_data else '<div class="chart-empty muted">No telemetry data available</div>'
    temp_chart_empty = "" if temp_data else '<div class="chart-empty muted">No telemetry data available</div>'
    bat_canvas_style = "" if battery_data else 'style="display:none"'
    temp_canvas_style = "" if temp_data else 'style="display:none"'

    # Alert banner if battery critical
    alert_html = ""
    if bat_level is not None and bat_level <= 10:
        alert_html = f'''
        <div class="alert alert-critical">
          <span class="alert-icon">⚠</span>
          Battery critically low ({bat_level}%) — device may be offline or about to die.
        </div>'''

    # android version from api level
    api_to_android = {
        33: "13", 32: "12L", 31: "12", 30: "11", 29: "10",
        28: "9 Pie", 27: "8.1", 26: "8.0",
    }
    android_ver = api_to_android.get(api_level, f"API {api_level}")

    html = f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Device Report — {name}</title>
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js"></script>
  <style>
    *, *::before, *::after {{ box-sizing: border-box; margin: 0; padding: 0; }}
    body {{
      background: #0a0f1e;
      color: #e2e8f0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      font-size: 14px;
      line-height: 1.5;
      padding: 2rem;
      min-height: 100vh;
    }}
    a {{ color: #38bdf8; text-decoration: none; }}

    /* ── Layout ── */
    .page {{ max-width: 1200px; margin: 0 auto; }}

    /* ── Header ── */
    .header {{
      display: flex; justify-content: space-between; align-items: flex-start;
      margin-bottom: 1.75rem; flex-wrap: wrap; gap: 1rem;
    }}
    .header-left h1 {{
      font-size: 2rem; font-weight: 700; letter-spacing: -0.02em;
      background: linear-gradient(135deg, #e2e8f0 0%, #94a3b8 100%);
      -webkit-background-clip: text; -webkit-text-fill-color: transparent;
    }}
    .header-left .sub {{
      color: #64748b; margin-top: 0.35rem; font-size: 0.85rem;
    }}
    .header-right {{ text-align: right; }}
    .header-right .gen-time {{ color: #475569; font-size: 0.8rem; margin-top: 0.35rem; }}

    /* ── Badges ── */
    .badge-green, .badge-red, .badge-muted {{
      display: inline-block; padding: 0.25rem 0.75rem;
      border-radius: 9999px; font-size: 0.75rem; font-weight: 600;
      letter-spacing: 0.05em;
    }}
    .badge-green  {{ background: rgba(74,222,128,0.15); color: #4ade80; border: 1px solid rgba(74,222,128,0.3); }}
    .badge-red    {{ background: rgba(248,113,113,0.15); color: #f87171; border: 1px solid rgba(248,113,113,0.3); }}
    .badge-muted  {{ background: rgba(148,163,184,0.1); color: #94a3b8; border: 1px solid rgba(148,163,184,0.2); }}

    /* ── Alert ── */
    .alert {{
      display: flex; align-items: center; gap: 0.75rem;
      padding: 0.75rem 1rem; border-radius: 0.5rem;
      margin-bottom: 1.5rem; font-size: 0.85rem; font-weight: 500;
    }}
    .alert-critical {{
      background: rgba(248,113,113,0.12); border: 1px solid rgba(248,113,113,0.3); color: #fca5a5;
    }}
    .alert-icon {{ font-size: 1rem; }}

    /* ── Cards ── */
    .cards-grid {{
      display: grid;
      grid-template-columns: repeat(5, 1fr);
      gap: 1rem;
      margin-bottom: 1.5rem;
    }}
    @media (max-width: 900px)  {{ .cards-grid {{ grid-template-columns: repeat(3, 1fr); }} }}
    @media (max-width: 600px)  {{ .cards-grid {{ grid-template-columns: repeat(2, 1fr); }} }}

    .card {{
      background: #131929; border: 1px solid #1e2d45;
      border-radius: 0.75rem; padding: 1.1rem 1.25rem;
      position: relative; overflow: hidden;
    }}
    .card-label {{
      font-size: 0.7rem; font-weight: 600; letter-spacing: 0.08em;
      text-transform: uppercase; color: #475569; margin-bottom: 0.5rem;
    }}
    .card-value {{
      font-size: 1.75rem; font-weight: 700; letter-spacing: -0.02em;
      line-height: 1;
    }}
    .card-sub {{
      font-size: 0.75rem; color: #475569; margin-top: 0.35rem;
    }}
    .card-icon {{
      position: absolute; top: 1rem; right: 1rem;
      font-size: 1.5rem; opacity: 0.15;
    }}

    /* battery card progress bar */
    .bat-bar-wrap {{
      background: #1e293b; border-radius: 9999px; height: 4px;
      margin-top: 0.75rem; overflow: hidden;
    }}
    .bat-bar-fill {{
      height: 100%; border-radius: 9999px;
      transition: width 0.5s ease;
    }}

    /* signal dots */
    .dot-filled {{ color: #38bdf8; }}
    .dot-empty  {{ color: #1e293b; }}

    /* ── Charts row ── */
    .charts-row {{
      display: grid; grid-template-columns: 3fr 2fr;
      gap: 1rem; margin-bottom: 1.5rem;
    }}
    @media (max-width: 700px) {{ .charts-row {{ grid-template-columns: 1fr; }} }}

    .chart-card {{
      background: #131929; border: 1px solid #1e2d45;
      border-radius: 0.75rem; padding: 1.25rem;
    }}
    .chart-title {{
      font-size: 0.75rem; font-weight: 600; letter-spacing: 0.06em;
      text-transform: uppercase; color: #475569; margin-bottom: 1rem;
    }}
    .chart-wrap {{ position: relative; height: 200px; }}
    .chart-empty {{
      display: flex; align-items: center; justify-content: center;
      height: 200px; font-size: 0.85rem;
    }}

    /* ── Info grid (below charts) ── */
    .info-grid {{
      display: grid; grid-template-columns: repeat(2, 1fr);
      gap: 1rem; margin-bottom: 1.5rem;
    }}
    @media (max-width: 600px) {{ .info-grid {{ grid-template-columns: 1fr; }} }}

    /* ── Commands table ── */
    .commands-card {{
      background: #131929; border: 1px solid #1e2d45;
      border-radius: 0.75rem; padding: 1.25rem;
    }}
    .section-title {{
      font-size: 0.75rem; font-weight: 600; letter-spacing: 0.06em;
      text-transform: uppercase; color: #475569; margin-bottom: 1rem;
    }}
    table {{ width: 100%; border-collapse: collapse; }}
    th {{
      text-align: left; font-size: 0.7rem; font-weight: 600;
      letter-spacing: 0.06em; text-transform: uppercase; color: #334155;
      padding: 0 0.75rem 0.6rem;
      border-bottom: 1px solid #1e2d45;
    }}
    td {{
      padding: 0.6rem 0.75rem; border-bottom: 1px solid #0f1929;
      font-size: 0.83rem; vertical-align: middle;
    }}
    tr:last-child td {{ border-bottom: none; }}
    code {{
      font-family: "SF Mono", "Fira Code", Menlo, monospace;
      font-size: 0.78rem; color: #e2e8f0;
    }}
    .td-date  {{ color: #475569; white-space: nowrap; }}
    .td-empty {{ text-align: center; padding: 2rem; }}

    .cmd-badge {{
      display: inline-block; padding: 0.15rem 0.5rem;
      border-radius: 4px; font-size: 0.7rem; font-weight: 600;
      letter-spacing: 0.04em;
    }}
    .cmd-success {{ background: rgba(74,222,128,0.12); color: #4ade80; }}
    .cmd-fail    {{ background: rgba(248,113,113,0.12); color: #f87171; }}
    .cmd-neutral {{ background: rgba(148,163,184,0.1); color: #94a3b8; }}

    /* ── Device info card ── */
    .kv-list {{ list-style: none; }}
    .kv-list li {{
      display: flex; justify-content: space-between; align-items: center;
      padding: 0.45rem 0; border-bottom: 1px solid #0f1929;
      font-size: 0.83rem;
    }}
    .kv-list li:last-child {{ border-bottom: none; }}
    .kv-key {{ color: #475569; }}
    .kv-val {{ font-weight: 500; text-align: right; max-width: 60%; word-break: break-all; }}

    /* ── Footer ── */
    .footer {{
      margin-top: 2rem; padding-top: 1rem; border-top: 1px solid #1e2d45;
      color: #1e293b; font-size: 0.75rem; text-align: center;
    }}

    .muted {{ color: #475569; }}
  </style>
</head>
<body>
<div class="page">

  <!-- Header -->
  <div class="header">
    <div class="header-left">
      <h1>{name}</h1>
      <div class="sub">{hw_name} &nbsp;·&nbsp; Android {android_ver} &nbsp;·&nbsp; GMS: {is_gms}</div>
      <div class="sub" style="margin-top:0.2rem; word-break:break-all; font-size:0.78rem; color:#334155">{dev_id}</div>
    </div>
    <div class="header-right">
      {_state_badge(state_val)}
      <div class="gen-time">Generated {generated}</div>
    </div>
  </div>

  {alert_html}

  <!-- Metric cards -->
  <div class="cards-grid">

    <!-- Battery -->
    <div class="card" style="background:{bat_bg}; border-color: {bat_color}33;">
      <div class="card-label">Battery</div>
      <div class="card-value" style="color:{bat_color}">{bat_pct_display}</div>
      <div class="card-sub">{bat_temp_display} temperature</div>
      <div class="bat-bar-wrap">
        <div class="bat-bar-fill" style="width:{bat_pct_val}%; background:{bat_color}"></div>
      </div>
    </div>

    <!-- Temperature -->
    <div class="card">
      <div class="card-label">Temperature</div>
      <div class="card-value" style="color:#38bdf8">{bat_temp_display}</div>
      <div class="card-sub">device body temp</div>
      <div class="card-icon">🌡</div>
    </div>

    <!-- Storage -->
    <div class="card">
      <div class="card-label">Free Storage</div>
      <div class="card-value" style="color:#a78bfa">{_mb_display(storage_mb)}</div>
      <div class="card-sub">available space</div>
      <div class="card-icon">💾</div>
    </div>

    <!-- RAM -->
    <div class="card">
      <div class="card-label">Free RAM</div>
      <div class="card-value" style="color:#67e8f9">{_mb_display(ram_mb)}</div>
      <div class="card-sub">available memory</div>
      <div class="card-icon">⚡</div>
    </div>

    <!-- WiFi -->
    <div class="card">
      <div class="card-label">WiFi</div>
      <div class="card-value" style="font-size:1.1rem; padding-top:0.25rem">
        {_signal_dots(sig_str)}
      </div>
      <div class="card-sub">{link_display} link speed</div>
      <div class="card-icon">📶</div>
    </div>

  </div>

  <!-- Charts -->
  <div class="charts-row">
    <div class="chart-card">
      <div class="chart-title">Battery Level — 30 Day Trend</div>
      <div class="chart-wrap">
        {bat_chart_empty}
        <canvas id="batteryChart" {bat_canvas_style}></canvas>
      </div>
    </div>
    <div class="chart-card">
      <div class="chart-title">Temperature — 30 Day Trend</div>
      <div class="chart-wrap">
        {temp_chart_empty}
        <canvas id="tempChart" {temp_canvas_style}></canvas>
      </div>
    </div>
  </div>

  <!-- Info + Commands -->
  <div class="info-grid">

    <!-- Device info -->
    <div class="card">
      <div class="section-title">Device Info</div>
      <ul class="kv-list">
        <li><span class="kv-key">Hardware</span><span class="kv-val">{hw_name}</span></li>
        <li><span class="kv-key">Android</span><span class="kv-val">{android_ver} (API {api_level})</span></li>
        <li><span class="kv-key">GMS</span><span class="kv-val">{is_gms}</span></li>
        <li><span class="kv-key">State</span><span class="kv-val">{state_val}</span></li>
        <li><span class="kv-key">Installed Apps</span><span class="kv-val">{installs_count}</span></li>
        <li><span class="kv-key">Device ID</span>
          <span class="kv-val" style="font-family:monospace; font-size:0.72rem">{dev_id}</span>
        </li>
      </ul>
    </div>

    <!-- Recent commands -->
    <div class="commands-card">
      <div class="section-title">Recent Commands</div>
      <table>
        <thead>
          <tr>
            <th>Date</th>
            <th>Command</th>
            <th>Issued By</th>
            <th>State</th>
          </tr>
        </thead>
        <tbody>
          {cmd_rows}
        </tbody>
      </table>
    </div>

  </div>

  <div class="footer">espercli device report &nbsp;·&nbsp; {generated}</div>

</div><!-- /page -->

<script>
  // ── Injected data ──────────────────────────────────────────────────────
  const batLabels  = {bat_labels};
  const batValues  = {bat_values};
  const tempLabels = {temp_labels};
  const tempValues = {temp_values};
  const batColor   = "{bat_color}";

  // ── Helper: gradient fill ──────────────────────────────────────────────
  function makeGradient(ctx, color) {{
    const g = ctx.createLinearGradient(0, 0, 0, 200);
    g.addColorStop(0, color + "55");
    g.addColorStop(1, color + "00");
    return g;
  }}

  // ── Battery chart ──────────────────────────────────────────────────────
  if (batLabels.length > 0) {{
    const batCtx = document.getElementById("batteryChart").getContext("2d");
    new Chart(batCtx, {{
      type: "line",
      data: {{
        labels: batLabels,
        datasets: [{{
          label: "Battery %",
          data: batValues,
          borderColor: batColor,
          backgroundColor: makeGradient(batCtx, batColor),
          borderWidth: 2,
          pointRadius: 3,
          pointBackgroundColor: batColor,
          tension: 0.35,
          fill: true,
        }}]
      }},
      options: {{
        responsive: true,
        maintainAspectRatio: false,
        scales: {{
          x: {{
            grid: {{ color: "#1e2d45" }},
            ticks: {{ color: "#475569", font: {{ size: 10 }}, maxTicksLimit: 8 }}
          }},
          y: {{
            min: 0, max: 100,
            grid: {{ color: "#1e2d45" }},
            ticks: {{ color: "#475569", font: {{ size: 10 }}, callback: v => v + "%" }}
          }}
        }},
        plugins: {{
          legend: {{ display: false }},
          tooltip: {{
            backgroundColor: "#1e293b",
            borderColor: "#334155",
            borderWidth: 1,
            titleColor: "#94a3b8",
            bodyColor: "#e2e8f0",
            callbacks: {{ label: ctx => " " + ctx.parsed.y.toFixed(1) + "%" }}
          }}
        }}
      }}
    }});
  }}

  // ── Temperature chart ──────────────────────────────────────────────────
  if (tempLabels.length > 0) {{
    const tempCtx = document.getElementById("tempChart").getContext("2d");
    new Chart(tempCtx, {{
      type: "line",
      data: {{
        labels: tempLabels,
        datasets: [{{
          label: "Temp °C",
          data: tempValues,
          borderColor: "#38bdf8",
          backgroundColor: makeGradient(tempCtx, "#38bdf8"),
          borderWidth: 2,
          pointRadius: 3,
          pointBackgroundColor: "#38bdf8",
          tension: 0.35,
          fill: true,
        }}]
      }},
      options: {{
        responsive: true,
        maintainAspectRatio: false,
        scales: {{
          x: {{
            grid: {{ color: "#1e2d45" }},
            ticks: {{ color: "#475569", font: {{ size: 10 }}, maxTicksLimit: 6 }}
          }},
          y: {{
            grid: {{ color: "#1e2d45" }},
            ticks: {{ color: "#475569", font: {{ size: 10 }}, callback: v => v + "°" }}
          }}
        }},
        plugins: {{
          legend: {{ display: false }},
          tooltip: {{
            backgroundColor: "#1e293b",
            borderColor: "#334155",
            borderWidth: 1,
            titleColor: "#94a3b8",
            bodyColor: "#e2e8f0",
            callbacks: {{ label: ctx => " " + ctx.parsed.y.toFixed(1) + "°C" }}
          }}
        }}
      }}
    }});
  }}
</script>
</body>
</html>"""

    return html
