package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type EventEntry struct {
	Event      int    `json:"event"`
	Platform   string `json:"platform"`
	Action     string `json:"action"`
	Element    string `json:"element,omitempty"`
	X          int    `json:"x,omitempty"`
	Y          int    `json:"y,omitempty"`
	Status     string `json:"status"`
	Time       string `json:"time,omitempty"`
	Screenshot bool   `json:"screenshot"`
}

type CrashEntry struct {
	Event      int    `json:"event"`
	Message    string `json:"message"`
	Screenshot string `json:"screenshot,omitempty"`
	LogSnippet string `json:"log_snippet,omitempty"`
}

type Report struct {
	Dir          string
	Events       []EventEntry
	Crashes      []CrashEntry
	Screenshots  []string
	LogLines     []string
	StartTime    time.Time
	EndTime      time.Time
	TotalEvents  int
	TotalCrashes int
	Platform     string
	DeviceName   string
	// ClosestScreenshot maps event number → screenshot filename for display.
	ClosestScreenshot map[int]string
}

func (r *Report) WriteEventsJSON() error {
	f, err := os.Create(filepath.Join(r.Dir, "events.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(r.Events)
}

func (r *Report) WriteLogs() error {
	logDir := filepath.Join(r.Dir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(logDir, "crash.log"), []byte(strings.Join(r.LogLines, "\n")), 0644)
}

func (r *Report) AddScreenshot(filename string) {
	r.Screenshots = append(r.Screenshots, filename)
}

func (r *Report) WriteHTML() error {
	return os.WriteFile(filepath.Join(r.Dir, "index.html"), []byte(r.html()), 0644)
}

func (r *Report) html() string {
	duration := r.EndTime.Sub(r.StartTime).Round(time.Second)
	passCount := 0
	failCount := 0
	actionCounts := map[string]int{}
	for _, e := range r.Events {
		if e.Status == "ok" {
			passCount++
		} else {
			failCount++
		}
		actionCounts[e.Action]++
	}
	passRate := float64(0)
	if len(r.Events) > 0 {
		passRate = float64(passCount) / float64(len(r.Events)) * 100
	}

	sb := &strings.Builder{}
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Monkeyrun Report</title>
<style>
:root{--bg:#1a1a2e;--surface:#16213e;--surface2:#0f3460;--border:#1a3a5c;--text:#e4e4e7;--text2:#94a3b8;--accent:#e94560;--green:#22c55e;--red:#ef4444;--yellow:#eab308;--blue:#3b82f6;--purple:#a855f7;--cyan:#06b6d4;--orange:#f97316;--radius:8px}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Inter',system-ui,-apple-system,sans-serif;background:var(--bg);color:var(--text);display:flex;min-height:100vh}
.sidebar{width:260px;background:var(--surface);border-right:1px solid var(--border);padding:1.25rem;display:flex;flex-direction:column;position:fixed;top:0;left:0;bottom:0;overflow-y:auto}
.sidebar h1{font-size:1rem;font-weight:700;letter-spacing:-.02em;margin-bottom:.25rem}
.sidebar .subtitle{font-size:.7rem;color:var(--text2);margin-bottom:1.5rem;text-transform:uppercase;letter-spacing:.05em}
.nav-section{margin-bottom:1.25rem}
.nav-section h3{font-size:.65rem;text-transform:uppercase;letter-spacing:.08em;color:var(--text2);margin-bottom:.5rem;font-weight:600}
.nav-item{display:flex;align-items:center;gap:.5rem;padding:.45rem .6rem;border-radius:var(--radius);cursor:pointer;font-size:.8rem;color:var(--text2);transition:all .15s;text-decoration:none}
.nav-item:hover,.nav-item.active{background:var(--surface2);color:var(--text)}
.nav-item .count{margin-left:auto;background:var(--surface2);padding:.1rem .45rem;border-radius:99px;font-size:.65rem;font-weight:600}
.nav-item .dot{width:8px;height:8px;border-radius:50%;flex-shrink:0}
.dot-green{background:var(--green)}.dot-red{background:var(--red)}.dot-yellow{background:var(--yellow)}
.main{margin-left:260px;flex:1;padding:1.5rem 2rem;max-width:1100px}
.header{margin-bottom:1.75rem}
.header h2{font-size:1.35rem;font-weight:700;margin-bottom:.35rem}
.header .meta{font-size:.75rem;color:var(--text2);display:flex;gap:1rem;flex-wrap:wrap}
.meta-chip{background:var(--surface);padding:.2rem .6rem;border-radius:99px;border:1px solid var(--border)}
.stats{display:grid;grid-template-columns:repeat(auto-fill,minmax(155px,1fr));gap:.75rem;margin-bottom:1.75rem}
.stat{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:.85rem 1rem}
.stat .val{font-size:1.6rem;font-weight:700;letter-spacing:-.02em}
.stat .lbl{font-size:.65rem;color:var(--text2);text-transform:uppercase;letter-spacing:.06em;margin-top:.15rem}
.stat.green .val{color:var(--green)}.stat.red .val{color:var(--red)}.stat.blue .val{color:var(--blue)}
.progress-bar{height:6px;background:var(--surface2);border-radius:99px;overflow:hidden;margin-bottom:1.75rem}
.progress-fill{height:100%;border-radius:99px;transition:width .3s}
.section{margin-bottom:2rem}
.section h3{font-size:.9rem;font-weight:600;margin-bottom:.75rem;display:flex;align-items:center;gap:.5rem}
.section h3 .badge{font-size:.65rem;background:var(--surface2);padding:.15rem .5rem;border-radius:99px;color:var(--text2);font-weight:500}
.filter-bar{display:flex;gap:.4rem;margin-bottom:.75rem;flex-wrap:wrap}
.filter-btn{padding:.3rem .7rem;border-radius:99px;border:1px solid var(--border);background:transparent;color:var(--text2);font-size:.7rem;cursor:pointer;transition:all .15s}
.filter-btn:hover,.filter-btn.active{background:var(--surface2);color:var(--text);border-color:var(--accent)}
table{width:100%;border-collapse:collapse;font-size:.78rem}
thead th{text-align:left;padding:.55rem .65rem;color:var(--text2);font-weight:500;font-size:.65rem;text-transform:uppercase;letter-spacing:.06em;border-bottom:1px solid var(--border);position:sticky;top:0;background:var(--bg)}
tbody td{padding:.5rem .65rem;border-bottom:1px solid rgba(255,255,255,.04)}
tbody tr{transition:background .1s}
tbody tr:hover{background:rgba(255,255,255,.03)}
.action-badge{display:inline-block;padding:.15rem .5rem;border-radius:99px;font-size:.65rem;font-weight:600;text-transform:uppercase;letter-spacing:.03em}
.badge-tap{background:rgba(59,130,246,.15);color:var(--blue)}
.badge-doubleTap{background:rgba(168,85,247,.15);color:var(--purple)}
.badge-longPress{background:rgba(249,115,22,.15);color:var(--orange)}
.badge-swipe,.badge-scroll{background:rgba(6,182,212,.15);color:var(--cyan)}
.badge-type{background:rgba(234,179,8,.15);color:var(--yellow)}
.badge-back{background:rgba(148,163,184,.15);color:var(--text2)}
.badge-error,.badge-hierarchy{background:rgba(239,68,68,.15);color:var(--red)}
.status-pass{color:var(--green);font-weight:600}.status-fail{color:var(--red);font-weight:600}
.element-name{color:var(--text2);font-family:'SF Mono',Consolas,monospace;font-size:.72rem}
.timeline-wrap{max-height:520px;overflow-y:auto;border:1px solid var(--border);border-radius:var(--radius)}
.crash-card{background:var(--surface);border:1px solid var(--border);border-left:3px solid var(--red);border-radius:var(--radius);padding:.85rem 1rem;margin-bottom:.6rem}
.crash-card .crash-head{display:flex;justify-content:space-between;align-items:center;margin-bottom:.35rem}
.crash-card .crash-event{font-size:.7rem;background:rgba(239,68,68,.15);color:var(--red);padding:.1rem .5rem;border-radius:99px;font-weight:600}
.crash-card .crash-msg{font-size:.75rem;color:var(--text2);font-family:'SF Mono',Consolas,monospace;word-break:break-all}
.screenshots-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(180px,1fr));gap:.6rem}
.screenshot-card{border:1px solid var(--border);border-radius:var(--radius);overflow:hidden;transition:transform .15s}
.screenshot-card:hover{transform:scale(1.02)}
.screenshot-card img{width:100%;display:block}
.screenshot-card .cap{padding:.35rem .5rem;font-size:.65rem;color:var(--text2);background:var(--surface)}
pre.log-box{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:1rem;font-size:.7rem;color:var(--text2);max-height:320px;overflow:auto;white-space:pre-wrap;word-break:break-all;font-family:'SF Mono',Consolas,monospace;line-height:1.6}
.empty{color:var(--text2);font-size:.8rem;font-style:italic;padding:1rem}
@media(max-width:768px){.sidebar{display:none}.main{margin-left:0}}
</style>
</head>
<body>
<aside class="sidebar">
  <h1>Monkeyrun</h1>
  <div class="subtitle">Chaos Test Report</div>
  <nav>
    <div class="nav-section">
      <h3>Overview</h3>
      <a href="#summary" class="nav-item active"><div class="dot dot-green"></div>Summary</a>
      <a href="#timeline" class="nav-item"><div class="dot dot-yellow"></div>Timeline<span class="count">`)
	fmt.Fprintf(sb, "%d", len(r.Events))
	sb.WriteString(`</span></a>
    </div>
    <div class="nav-section">
      <h3>Results</h3>
      <a href="#crashes" class="nav-item"><div class="dot dot-red"></div>Crashes<span class="count">`)
	fmt.Fprintf(sb, "%d", len(r.Crashes))
	sb.WriteString(`</span></a>
      <a href="#screenshots" class="nav-item">Screenshots<span class="count">`)
	fmt.Fprintf(sb, "%d", len(r.Screenshots))
	sb.WriteString(`</span></a>
      <a href="#logs" class="nav-item">Logs</a>
    </div>
    <div class="nav-section">
      <h3>Actions</h3>`)
	for _, act := range []string{"tap", "doubleTap", "longPress", "swipe", "scroll", "type", "back"} {
		if c, ok := actionCounts[act]; ok && c > 0 {
			fmt.Fprintf(sb, `<div class="nav-item"><span class="action-badge badge-%s">%s</span><span class="count">%d</span></div>`, act, act, c)
		}
	}
	sb.WriteString(`
    </div>
  </nav>
</aside>
<main class="main">
  <div class="header" id="summary">
    <h2>Test Run Results</h2>
    <div class="meta">`)
	if r.DeviceName != "" {
		fmt.Fprintf(sb, `<span class="meta-chip">%s</span>`, esc(r.DeviceName))
	}
	if r.Platform != "" {
		fmt.Fprintf(sb, `<span class="meta-chip">%s</span>`, esc(r.Platform))
	}
	fmt.Fprintf(sb, `<span class="meta-chip">%s</span>`, duration)
	fmt.Fprintf(sb, `<span class="meta-chip">%s</span>`, r.EndTime.Format("Jan 2, 2006 15:04:05"))
	sb.WriteString(`</div>
  </div>
  <div class="stats">
    <div class="stat"><div class="val">`)
	fmt.Fprintf(sb, "%d", r.TotalEvents)
	sb.WriteString(`</div><div class="lbl">Total Events</div></div>
    <div class="stat green"><div class="val">`)
	fmt.Fprintf(sb, "%d", passCount)
	sb.WriteString(`</div><div class="lbl">Passed</div></div>
    <div class="stat red"><div class="val">`)
	fmt.Fprintf(sb, "%d", failCount)
	sb.WriteString(`</div><div class="lbl">Failed</div></div>
    <div class="stat red"><div class="val">`)
	fmt.Fprintf(sb, "%d", r.TotalCrashes)
	sb.WriteString(`</div><div class="lbl">Crashes</div></div>
    <div class="stat blue"><div class="val">`)
	fmt.Fprintf(sb, "%.0f%%", passRate)
	sb.WriteString(`</div><div class="lbl">Pass Rate</div></div>
  </div>
  <div class="progress-bar"><div class="progress-fill" style="width:`)
	fmt.Fprintf(sb, "%.1f", passRate)
	sb.WriteString(`%;background:`)
	if passRate >= 80 {
		sb.WriteString("var(--green)")
	} else if passRate >= 50 {
		sb.WriteString("var(--yellow)")
	} else {
		sb.WriteString("var(--red)")
	}
	sb.WriteString(`"></div></div>

  <div class="section" id="timeline">
    <h3>Timeline <span class="badge">`)
	fmt.Fprintf(sb, "%d events", len(r.Events))
	sb.WriteString(`</span></h3>
    <div class="filter-bar">
      <button class="filter-btn active" onclick="filterRows('all')">All</button>
      <button class="filter-btn" onclick="filterRows('ok')">Passed</button>
      <button class="filter-btn" onclick="filterRows('fail')">Failed</button>`)
	for _, act := range []string{"tap", "doubleTap", "longPress", "swipe", "scroll", "type", "back"} {
		if actionCounts[act] > 0 {
			fmt.Fprintf(sb, `<button class="filter-btn" onclick="filterRows('%s')">%s</button>`, act, act)
		}
	}
	sb.WriteString(`
    </div>
    <div class="timeline-wrap">
      <table id="events-table">
        <thead><tr><th>#</th><th>Time</th><th>Action</th><th>Element</th><th>Coords</th><th>Status</th><th></th></tr></thead>
        <tbody>`)
	for _, e := range r.Events {
		statusCls := "status-pass"
		statusTxt := "passed"
		if e.Status != "ok" {
			statusCls = "status-fail"
			statusTxt = esc(e.Status)
		}
		timeShort := e.Time
		if len(timeShort) > 19 {
			timeShort = timeShort[11:19]
		}
		fmt.Fprintf(sb, `<tr data-status="%s" data-action="%s">`, e.Status, e.Action)
		fmt.Fprintf(sb, `<td>%d</td>`, e.Event)
		fmt.Fprintf(sb, `<td>%s</td>`, esc(timeShort))
		fmt.Fprintf(sb, `<td><span class="action-badge badge-%s">%s</span></td>`, e.Action, esc(e.Action))
		fmt.Fprintf(sb, `<td class="element-name">%s</td>`, esc(e.Element))
		if e.X != 0 || e.Y != 0 {
			fmt.Fprintf(sb, `<td class="element-name">%d,%d</td>`, e.X, e.Y)
		} else {
			sb.WriteString(`<td></td>`)
		}
		fmt.Fprintf(sb, `<td class="%s">%s</td>`, statusCls, statusTxt)
		if e.Screenshot {
			ssName := ""
			if r.ClosestScreenshot != nil {
				ssName = r.ClosestScreenshot[e.Event]
			}
			if ssName != "" {
				fmt.Fprintf(sb, `<td><a href="screenshots/%s" title="View screenshot" style="color:var(--cyan);text-decoration:none">&#128247;</a></td>`, esc(ssName))
			} else {
				sb.WriteString(`<td style="color:var(--cyan)">&#128247;</td>`)
			}
		} else {
			sb.WriteString(`<td></td>`)
		}
		sb.WriteString(`</tr>`)
	}
	sb.WriteString(`</tbody></table>
    </div>
  </div>

  <div class="section" id="crashes">
    <h3>Crashes <span class="badge">`)
	fmt.Fprintf(sb, "%d", len(r.Crashes))
	sb.WriteString(`</span></h3>`)
	if len(r.Crashes) == 0 {
		sb.WriteString(`<div class="empty">No crashes detected.</div>`)
	}
	for _, c := range r.Crashes {
		sb.WriteString(`<div class="crash-card"><div class="crash-head">`)
		fmt.Fprintf(sb, `<span class="crash-event">Event #%d</span>`, c.Event)
		sb.WriteString(`</div>`)
		fmt.Fprintf(sb, `<div class="crash-msg">%s</div>`, esc(c.Message))
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>

  <div class="section" id="screenshots">
    <h3>Screenshots <span class="badge">`)
	fmt.Fprintf(sb, "%d", len(r.Screenshots))
	sb.WriteString(`</span></h3>`)
	if len(r.Screenshots) == 0 {
		sb.WriteString(`<div class="empty">No screenshots captured.</div>`)
	} else {
		sb.WriteString(`<div class="screenshots-grid">`)
		for _, name := range r.Screenshots {
			sb.WriteString(`<div class="screenshot-card">`)
			fmt.Fprintf(sb, `<a href="screenshots/%s"><img src="screenshots/%s" loading="lazy"></a>`, esc(name), esc(name))
			fmt.Fprintf(sb, `<div class="cap">%s</div>`, esc(name))
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>

  <div class="section" id="logs">
    <h3>Logs</h3>`)
	if len(r.LogLines) == 0 {
		sb.WriteString(`<div class="empty">No logs captured.</div>`)
	} else {
		sb.WriteString(`<pre class="log-box">`)
		sb.WriteString(esc(strings.Join(r.LogLines, "\n")))
		sb.WriteString(`</pre>`)
	}
	sb.WriteString(`</div>
</main>
<script>
function filterRows(f){
  document.querySelectorAll('.filter-btn').forEach(b=>b.classList.remove('active'));
  event.target.classList.add('active');
  document.querySelectorAll('#events-table tbody tr').forEach(r=>{
    if(f==='all'){r.style.display='';}
    else if(f==='ok'){r.style.display=r.dataset.status==='ok'?'':'none';}
    else if(f==='fail'){r.style.display=r.dataset.status!=='ok'?'':'none';}
    else{r.style.display=r.dataset.action===f?'':'none';}
  });
}
document.querySelectorAll('.sidebar .nav-item').forEach(a=>{
  a.addEventListener('click',()=>{
    document.querySelectorAll('.sidebar .nav-item').forEach(n=>n.classList.remove('active'));
    a.classList.add('active');
  });
});
</script>
</body>
</html>`)
	return sb.String()
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
