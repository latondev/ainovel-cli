# Kế hoạch: Web Dashboard điều khiển ainovel-cli từ xa

> **Mục tiêu:** Đẩy ainovel-cli lên VPS, dựng một web/API server để từ máy khác có thể:
> - Yêu cầu viết truyện mới (gửi prompt + chọn style/model).
> - Theo dõi **tiến trình real-time**: chương đang viết, chương đã xong, % hoàn thành, số chữ.
> - Xem **trạng thái cả bộ truyện**: premise, đề cương, hệ thống nhân vật, quy tắc thế giới, phục bút, timeline, review biên tập.
> - Quản lý **nhiều truyện cùng lúc**, mỗi truyện lưu riêng để check lại.
>
> Tài liệu này là bản thiết kế để thực thi sau. Khi cần làm, đọc lại file này.
> _Cập nhật lần đầu: 2026-06 (giai đoạn thiết kế)._

---

## 0. TL;DR — Quyết định chính

| Vấn đề | Quyết định |
|---|---|
| Ngôn ngữ backend | **Go** (cùng repo, dùng lại được `internal/domain` để parse JSON) |
| Frontend | **Go html/template + HTMX + SSE** (nhẹ, không cần Node/React, không build step) |
| DB | **SQLite** (`modernc.org/sqlite`, pure-Go, không cần CGO) — chỉ làm **index**, nội dung vẫn ở file |
| Cách chạy CLI | Orchestrator **spawn 1 process ainovel-cli / 1 truyện** qua `os/exec`, mỗi truyện 1 **thư mục làm việc riêng** |
| Real-time | **Watch file** `progress.json` (fsnotify) → đẩy qua **SSE** xuống browser |
| Đa truyện | Mỗi truyện = 1 working dir `data/novels/<slug>/` → CLI tự ghi vào `<slug>/output/novel/` |
| LLM | Cloud (DeepSeek/OpenRouter…) → VPS 2 core/4GB **đủ thoải mái** |
| Can thiệp real-time | **Phase 3** (V1 chỉ đọc + tạo truyện mới). Lý do: xem mục 8. |

---

## 1. Ràng buộc đã xác minh từ source code

Những điều dưới đây **đã đọc code để xác nhận**, không phải phỏng đoán:

1. **CLI có chế độ headless** — `cmd/ainovel-cli/main.go`:
   ```
   ainovel-cli --headless --prompt "yêu cầu truyện"
   ainovel-cli --headless --prompt-file path.txt
   ainovel-cli --headless --prompt-file -   # đọc stdin
   ainovel-cli --config /path/config.json --headless --prompt "..."
   ```
   → Orchestrator gọi CLI bằng đúng các flag này.

2. **Headless KHÔNG hỗ trợ first-time setup** (`main.go:47-49`). → `config.json` **phải tồn tại sẵn** trước khi spawn. Orchestrator phải đảm bảo điều này.

3. **OutputDir cố định, KHÔNG cấu hình qua config.json** — `internal/bootstrap/config.go:117`:
   ```go
   OutputDir string `json:"-"`   // json:"-" => không đọc từ file config
   ```
   Mặc định = `output/novel` **tương đối theo thư mục chạy process** (`config.go:308-309`).
   → **Hệ quả quan trọng:** muốn nhiều truyện song song mà KHÔNG sửa code CLI, phải cho mỗi process một **working directory (cwd) khác nhau**. Mỗi truyện ghi vào `<cwd>/output/novel/`.

4. **Resume tự động** — chạy lại CLI trong cùng thư mục sẽ khôi phục từ checkpoint cuối (README + `store.CheckConsistency`). → Orchestrator restart 1 truyện = chỉ cần spawn lại trong đúng working dir.

5. **Output đã có cấu trúc sẵn** (xác nhận trên `output/novel/` thật) — không cần parse phức tạp, đọc thẳng JSON. Xem mục 3.

6. **Style tự nạp trọn bộ genre** (`assets/load.go`) — đặt `"style":"kinh_di_vietnam"` trong config kéo cả `styles/`, `references/genres/.../`, `knowledge/*.md`. **Nhưng assets nhúng tĩnh (`go:embed`)** → thêm genre mới phải **rebuild** binary/Docker image.

---

## 2. Kiến trúc tổng thể

```
   [Máy bạn - trình duyệt]
            │  HTTPS
            ▼
   ┌─────────────────────────────────────────────┐
   │            VPS (Ubuntu 2c/4GB)               │
   │                                              │
   │  ┌────────────────────────────────────────┐  │
   │  │   ainovel-web  (Go server - MỚI)        │  │
   │  │  - REST API + SSE                       │  │
   │  │  - html/template + HTMX (UI)            │  │
   │  │  - Orchestrator (spawn/track process)   │  │
   │  │  - File watcher (fsnotify)              │  │
   │  └──────┬───────────────┬─────────┬────────┘  │
   │         │ os/exec       │ đọc file │ SQL       │
   │         ▼               ▼         ▼            │
   │  ┌────────────┐  ┌───────────┐ ┌──────────┐   │
   │  │ ainovel-cli│  │ data/novels│ │ index.db │   │
   │  │ (N process)│  │  /<slug>/  │ │ (SQLite) │   │
   │  └─────┬──────┘  └───────────┘ └──────────┘   │
   │        │ HTTPS                                 │
   │        ▼                                       │
   │   LLM Cloud (DeepSeek / OpenRouter)            │
   └─────────────────────────────────────────────┘
```

**Nguyên tắc:** `ainovel-web` là **vỏ điều phối + đọc dữ liệu**, KHÔNG sửa logic sáng tác của CLI. CLI vẫn là nguồn sự thật; web chỉ đọc file nó sinh ra và quản lý vòng đời process.

---

## 3. Các file dữ liệu CLI sinh ra (nguồn cho web)

Mỗi truyện ở `data/novels/<slug>/output/novel/`:

| File | Nội dung | Dùng cho màn hình |
|---|---|---|
| `meta/progress.json` | `novel_name`, `phase`, `current_chapter`, `total_chapters`, `completed_chapters[]`, `total_word_count`, `chapter_word_counts{}` | **Thanh tiến độ + trạng thái tổng** (file watch chính) |
| `meta/run.json` | style, provider, model của lần chạy | Header truyện |
| `meta/usage.json` | token tiêu thụ | Thống kê chi phí |
| `premise.md` | Tiền đề câu chuyện | Tab "Tổng quan" |
| `outline.json` / `outline.md` | Đề cương chương | Tab "Đề cương" |
| `layered_outline.json` | Đề cương phân tầng (tập/cung) cho truyện dài | Cây tập→cung→chương |
| `characters.json` / `.md` | Hồ sơ nhân vật | Tab "Nhân vật" |
| `world_rules.json` / `.md` | Quy tắc thế giới | Tab "Thế giới" |
| `foreshadow_ledger.json` | Sổ phục bút (đã gieo / đã thu) | Tab "Phục bút" |
| `relationship_state.json` | Quan hệ nhân vật | Tab "Quan hệ" (đồ thị) |
| `timeline.json` | Dòng thời gian | Tab "Timeline" |
| `chapters/NN.md` | **Bản thảo cuối** của chương đã xong | Đọc truyện |
| `drafts/NN.draft.md`, `NN.plan.json` | Nháp + kế hoạch chương đang viết | "Đang viết" / preview |
| `reviews/*` | Đánh giá 7 chiều của biên tập | Tab "Review" |
| `summaries/*` | Tóm tắt chương/cung/tập | Tóm tắt nhanh |
| `meta/checkpoints.jsonl` | Checkpoint cấp step | (Nội bộ, debug) |

> Tận dụng được `internal/domain/*.go` (Progress, Outline, Character, ...) để **unmarshal đúng struct** thay vì tự định nghĩa lại — giảm rủi ro lệch schema khi CLI nâng cấp.

---

## 4. Quản lý đa truyện (multi-tenancy)

Vì `OutputDir` cố định theo cwd (mục 1.3), layout trên VPS:

```
/opt/ainovel/
├── bin/ainovel-cli                 # binary CLI (đã build kèm genre)
├── ainovel-web                     # binary server mới
├── config/
│   └── config.json                 # config gốc (api_key, provider, style)
├── data/
│   ├── index.db                    # SQLite: danh mục truyện
│   └── novels/
│       ├── truyen-than/            # ← working dir của 1 truyện
│       │   ├── config.json         # (tùy chọn) override style/model riêng
│       │   └── output/novel/...    # CLI tự sinh ở đây
│       ├── vai-liem/
│       │   └── output/novel/...
│       └── <slug>/
└── logs/
    └── <slug>.log                  # stdout/stderr của từng process
```

**Tạo truyện mới** = orchestrator:
1. Sinh `slug` từ tên truyện (unique).
2. Tạo `data/novels/<slug>/`, copy/symlink `config.json` (có thể chèn `style` theo lựa chọn).
3. Ghi prompt ra `data/novels/<slug>/prompt.txt`.
4. `exec.Command(bin, "--config", cfg, "--headless", "--prompt-file", "prompt.txt")` với `cmd.Dir = data/novels/<slug>/`.
5. Lưu record vào SQLite (slug, tên, pid, trạng thái, thời điểm).
6. Đăng ký watcher lên `output/novel/meta/progress.json`.

---

## 5. SQLite schema (index — nhẹ)

DB **không lưu nội dung truyện**, chỉ lưu metadata để liệt kê/tìm nhanh & quản lý process.

```sql
CREATE TABLE novels (
  slug          TEXT PRIMARY KEY,      -- định danh + tên thư mục
  title         TEXT NOT NULL,         -- tên hiển thị (từ progress.novel_name)
  prompt        TEXT NOT NULL,         -- yêu cầu gốc
  style         TEXT,                  -- vd kinh_di_vietnam
  provider      TEXT,
  model         TEXT,
  status        TEXT NOT NULL,         -- queued|running|paused|done|error|stopped
  phase         TEXT,                  -- cache từ progress.json: architecting|writing|...
  current_chapter   INTEGER DEFAULT 0,
  total_chapters    INTEGER DEFAULT 0,
  completed_count   INTEGER DEFAULT 0,
  word_count        INTEGER DEFAULT 0,
  created_at    INTEGER NOT NULL,      -- unix ts
  updated_at    INTEGER NOT NULL,
  pid           INTEGER,               -- process hiện tại (0 nếu không chạy)
  last_error    TEXT
);

CREATE TABLE runs (                    -- lịch sử mỗi lần spawn (tùy chọn, để debug)
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  slug       TEXT NOT NULL REFERENCES novels(slug),
  started_at INTEGER NOT NULL,
  ended_at   INTEGER,
  exit_code  INTEGER,
  log_path   TEXT
);

CREATE INDEX idx_novels_status ON novels(status);
```

> Trạng thái "sự thật" vẫn nằm ở `progress.json`. DB chỉ là **bản cache** để list nhanh; một goroutine đồng bộ `progress.json → DB` mỗi khi file đổi.

---

## 6. REST API + SSE (endpoint)

Base path `/api`. Auth: xem mục 9.

| Method | Endpoint | Chức năng |
|---|---|---|
| `GET`  | `/api/novels` | Danh sách truyện (từ SQLite) + trạng thái |
| `POST` | `/api/novels` | Tạo truyện mới `{title, prompt, style?, model?}` → spawn process |
| `GET`  | `/api/novels/{slug}` | Chi tiết: progress + run meta |
| `GET`  | `/api/novels/{slug}/premise` | premise.md |
| `GET`  | `/api/novels/{slug}/outline` | outline.json (hoặc layered_outline.json) |
| `GET`  | `/api/novels/{slug}/characters` | characters.json |
| `GET`  | `/api/novels/{slug}/world` | world_rules.json |
| `GET`  | `/api/novels/{slug}/foreshadow` | foreshadow_ledger.json |
| `GET`  | `/api/novels/{slug}/timeline` | timeline.json |
| `GET`  | `/api/novels/{slug}/relationships` | relationship_state.json |
| `GET`  | `/api/novels/{slug}/chapters` | Danh sách chương (số, tiêu đề, word count, trạng thái) |
| `GET`  | `/api/novels/{slug}/chapters/{n}` | Nội dung 1 chương (markdown → HTML) |
| `GET`  | `/api/novels/{slug}/reviews` | reviews/* |
| `GET`  | `/api/novels/{slug}/usage` | usage.json (token/chi phí) |
| `POST` | `/api/novels/{slug}/stop` | Dừng process (SIGINT — CLI lưu checkpoint) |
| `POST` | `/api/novels/{slug}/resume` | Spawn lại trong working dir (tự resume) |
| `GET`  | `/api/novels/{slug}/events` | **SSE stream**: progress, log tail, sự kiện chương mới |
| `GET`  | `/api/novels/{slug}/export?fmt=txt\|epub` | Gọi CLI `/export` hoặc dùng `internal/host/exp` |
| `DELETE`| `/api/novels/{slug}` | Xóa truyện (xác nhận 2 lớp) |

**SSE payload mẫu** (`/events`):
```json
{ "type": "progress", "phase": "writing", "current": 24, "total": 57, "completed": 23, "words": 354829 }
{ "type": "chapter_done", "n": 24, "words": 14210 }
{ "type": "log", "line": "..." }
{ "type": "status", "status": "done" }
```

---

## 7. Real-time tiến trình (cơ chế)

1. **fsnotify** theo dõi `data/novels/<slug>/output/novel/meta/progress.json` và thư mục `chapters/`.
2. Khi `progress.json` đổi → đọc lại → so sánh → đẩy event SSE + cập nhật SQLite.
3. Khi có file `chapters/NN.md` mới → bắn `chapter_done`.
4. **stdout/stderr** của process được pipe vào `logs/<slug>.log` + tail 50 dòng cuối đẩy qua SSE (`type:"log"`).
5. Browser dùng `EventSource` (HTMX SSE extension hoặc JS thuần) cập nhật thanh tiến độ không cần reload.

> Phương án dự phòng nếu fsnotify lỗi trên 1 số filesystem (vd volume Docker): **poll** `progress.json` mỗi 2–3s. Rẻ vì file nhỏ.

---

## 8. Can thiệp real-time (Phase 3 — cần khảo sát thêm)

- TUI cho phép gõ can thiệp khi đang viết (README mục "Can thiệp thời gian thực"); cơ chế lưu qua `internal/store/directives.go` + `signals.go`.
- **Headless hiện là "fire-and-forget"** — không có kênh nhập liệu khi đang chạy.
- Hướng khả thi (chưa xác minh đủ, để Phase 3):
  - **8a.** Web ghi directive vào file mà coordinator đang chạy đọc được (`directives`/`signals` store) → cần kiểm tra coordinator có poll các file này trong vòng lặp headless không.
  - **8b.** Đơn giản hơn: web cho phép **stop → sửa directive → resume**. An toàn, không cần đụng CLI.
- **V1 chốt:** chỉ đọc + tạo + stop/resume. Can thiệp sống để sau.

---

## 9. Bảo mật & vận hành VPS

- **Auth:** tối thiểu 1 lớp — session cookie + mật khẩu, hoặc Basic Auth qua reverse proxy. Vì chỉ mình bạn dùng, không cần phức tạp.
- **TLS:** đặt sau **Caddy** (tự Let's Encrypt) hoặc **Nginx**. `ainovel-web` chỉ nghe `127.0.0.1:8080`, proxy lo HTTPS.
- **API key LLM:** chỉ nằm trong `config/config.json` trên VPS (chmod 600), không lộ ra frontend.
- **Giới hạn tài nguyên:** 2 core/4GB → nên giới hạn **số process song song** (vd tối đa 2–3 truyện viết cùng lúc) bằng một **semaphore/queue** trong orchestrator. Truyện thứ N+1 ở trạng thái `queued`.
- **systemd** quản lý `ainovel-web` (auto-restart). Process CLI là con của web; web chịu trách nhiệm reap + ghi exit code.
- **Sao lưu:** `data/` là toàn bộ tài sản → cron `tar`/rsync định kỳ.

---

## 10. Cấu trúc thư mục code (đề xuất)

Đặt server trong cùng repo để tái dùng `internal/domain`:

```
cmd/ainovel-web/main.go            # entrypoint server
internal/web/
├── server.go                      # router, middleware, mount SSE/static
├── api_novels.go                  # handlers REST
├── sse.go                         # hub SSE + broadcast
├── orchestrator.go                # spawn/stop/resume, semaphore, reap
├── watcher.go                     # fsnotify → events
├── indexdb.go                     # SQLite (modernc.org/sqlite)
├── reader.go                      # đọc & parse file output (dùng internal/domain)
├── slug.go                        # sinh slug unique
└── auth.go                        # session/password
web/
├── templates/                     # html/template
│   ├── layout.html
│   ├── novels_list.html
│   ├── novel_detail.html          # tabs: tổng quan/đề cương/nhân vật/...
│   └── chapter.html
└── static/                        # htmx.min.js, css, sse ext
```

**Phụ thuộc mới (tối thiểu):**
- `github.com/fsnotify/fsnotify` — watch file.
- `modernc.org/sqlite` — SQLite pure-Go (không CGO, build tĩnh dễ).
- (đã có) `github.com/yuin/goldmark` hoặc tương đương để render markdown→HTML; nếu chưa có thì thêm.
- HTMX: 1 file JS tĩnh, không cần build.

---

## 11. Lộ trình thực thi (phân giai đoạn)

### Phase 0 — Móng (đọc dữ liệu)
- [ ] `cmd/ainovel-web` + router cơ bản, phục vụ static + 1 trang.
- [ ] `reader.go`: parse `progress.json`, `premise.md`, `outline`, `characters` từ 1 working dir có sẵn (dùng chính `output/novel` hiện tại để test).
- [ ] Trang **list** + trang **chi tiết** (đọc tĩnh, chưa real-time).
- _Done khi:_ mở web thấy được truyện "Người Vẽ Hồn" đang có trong `output/`.

### Phase 1 — Đa truyện + tạo mới
- [ ] SQLite index + đồng bộ từ `progress.json`.
- [ ] `orchestrator.go`: tạo working dir, spawn `--headless --prompt-file`, lưu pid.
- [ ] Form "Tạo truyện mới" (title, prompt, chọn style).
- [ ] Semaphore giới hạn process song song.
- _Done khi:_ bấm nút tạo → truyện mới bắt đầu viết, hiện trong danh sách.

### Phase 2 — Real-time
- [ ] `watcher.go` (fsnotify) + `sse.go` hub.
- [ ] Thanh tiến độ live, "chương mới xong" tự hiện, tail log.
- [ ] Stop (SIGINT) / Resume (spawn lại).
- _Done khi:_ ngồi máy khác xem chương nhảy số real-time không cần F5.

### Phase 3 — Nâng cao
- [ ] Đọc chương (markdown render), tab nhân vật/thế giới/phục bút/timeline/review.
- [ ] Export TXT/EPUB qua web.
- [ ] (Khảo sát) Can thiệp real-time — mục 8.
- [ ] Auth + TLS + systemd + backup.

---

## 12. Rủi ro & lưu ý

| Rủi ro | Giảm thiểu |
|---|---|
| Genre mới không xuất hiện vì assets nhúng tĩnh | Rebuild CLI binary/Docker **sau khi** thêm genre; CI build kèm `assets/` |
| `OutputDir` không đổi được qua config | Dùng **working dir riêng/truyện** (đã chọn). _Tùy chọn:_ patch CLI thêm `--output-dir`/env nếu muốn gọn hơn |
| fsnotify không bắn trên volume Docker/mạng | Fallback **poll** progress.json 2–3s |
| 2 process viết cùng lúc làm nghẽn 4GB | Process CLI nhẹ khi dùng cloud LLM, nhưng vẫn **giới hạn song song = 2** cho an toàn |
| Headless chết giữa chừng (mất mạng LLM) | CLI có checkpoint → orchestrator phát hiện exit≠0 → đánh dấu `error`, cho **resume** 1 nút |
| Lệch schema khi CLI nâng cấp | Parse bằng `internal/domain` thay vì struct tự chế; bọc try-parse, degrade graceful |
| Lộ API key | Key chỉ ở `config.json` (chmod 600), không gửi xuống frontend |

---

## 13. Câu hỏi cần chốt trước khi code (khi quay lại)

1. **Sửa CLI hay không?** Nếu chấp nhận patch nhỏ thêm `--output-dir`/env, layout đa truyện gọn hơn (không cần cwid riêng). Mặc định plan này **không sửa CLI**.
2. **Mức auth** mong muốn: chỉ mật khẩu đơn giản, hay cần nhiều user?
3. **Giới hạn số truyện viết song song** trên VPS: đề xuất 2. OK chứ?
4. **UI**: HTMX server-rendered (nhẹ, đúng đề xuất) hay muốn SPA tách biệt sau này?
5. **Docker hay binary trần** cho `ainovel-web` trên VPS?

> Khi bạn trả lời 5 câu trên, có thể bắt đầu **Phase 0** ngay.
