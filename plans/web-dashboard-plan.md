# Kế hoạch: Web Dashboard điều khiển ainovel-cli từ xa

> **Mục tiêu:** Đẩy ainovel-cli lên VPS, dựng một web/API server để từ máy khác có thể:
> - Yêu cầu viết truyện mới (gửi prompt + chọn style/model).
> - Theo dõi **tiến trình real-time**: chương đang viết, chương đã xong, % hoàn thành, số chữ.
> - Xem **trạng thái cả bộ truyện**: premise, đề cương, hệ thống nhân vật, quy tắc thế giới, phục bút, timeline, review biên tập.
> - Quản lý **nhiều truyện cùng lúc**, mỗi truyện lưu riêng để check lại.
>
> Tài liệu này là bản thiết kế để thực thi sau. Khi cần làm, đọc lại file này.
> _Cập nhật lần đầu: 2026-06 (giai đoạn thiết kế)._
> _Sửa lần 2: 2026-06-30 — xác minh atomic-write (1.7), không-có-signal-handling (1.8); thêm concurrency ownership (10.1), slug an toàn (10.2), sanitize output (10.3); chốt 5 quyết định (13)._
> _Sửa lần 3: 2026-06-30 — chuẩn hóa `progress.phase == "complete"`; chốt stop là hard kill; thêm startup recovery (9.1), queue scheduler (10.1b), export qua `internal/host/exp` (10.4), xóa an toàn (10.5), checklist test invariant (11.1)._
> _Sửa lần 4: 2026-06-30 — đổi frontend từ Go template/HTMX sang **React + Vite + TypeScript SPA**; Go backend giữ REST API + SSE và serve static `dist`._

---

## 0. TL;DR — Quyết định chính

| Vấn đề | Quyết định |
|---|---|
| Ngôn ngữ backend | **Go** (cùng repo, dùng lại được `internal/domain` để parse JSON) |
| Frontend | **React + Vite + TypeScript SPA**; Go serve static `web/frontend/dist` sau build |
| DB | **SQLite** (`modernc.org/sqlite`, pure-Go, không cần CGO) — chỉ làm **index**, nội dung vẫn ở file |
| Cách chạy CLI | Orchestrator **spawn 1 process ainovel-cli / 1 truyện** qua `os/exec`, mỗi truyện 1 **thư mục làm việc riêng** |
| Real-time | **Watch thư mục** `meta/` (fsnotify, vì ghi atomic-rename — mục 1.7) + poll 5s dự phòng → **SSE** xuống browser |
| Đa truyện | Mỗi truyện = 1 working dir `data/novels/<slug>/` → CLI tự ghi vào `<slug>/output/novel/` |
| LLM | Cloud (DeepSeek/OpenRouter…) → VPS 2 core/4GB **đủ thoải mái** |
| Can thiệp real-time | **Phase 3** (V1 chỉ đọc + tạo truyện mới). Lý do: xem mục 8. |
| Export | Web gọi trực tiếp `internal/host/exp.Run`; `/export` hiện là lệnh TUI, không phải CLI subcommand |

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

7. **Mọi file output ghi ATOMIC** — `internal/store/io.go:36-64` (`WriteFileUnlocked`): ghi `*.tmp-*` → `Sync()` → `os.Rename`. → **Hệ quả:** web KHÔNG bao giờ đọc phải JSON rách (không có partial write). NHƯNG rename thay inode → **fsnotify phải watch thư mục `meta/`, KHÔNG add watch trực tiếp lên `progress.json`** (watch trên file sẽ chết sau lần rename đầu). Đây là cái bẫy thật sự của real-time.

8. **CLI KHÔNG bắt SIGINT/signal** — `grep signal.Notify` toàn repo = 0 kết quả; `consume()` (`internal/entry/headless/run.go:93`) không có nhánh tín hiệu, comment `run.go:52` xác nhận "trường hợp bị kill từ bên ngoài không đi qua defer". → **Hệ quả:** "stop" = **hard kill** (SIGTERM/SIGKILL), KHÔNG graceful. An toàn nhờ checkpoint atomic (mục 1.7) đã ghi từng chương xong; chỉ mất **chương đang viết dở** → resume làm lại. Defer cleanup/diag KHÔNG chạy khi bị kill — chấp nhận được, nhưng đừng kỳ vọng flush sạch.

9. **⚠️ "Tạo mới" và "resume" là HAI lệnh khác nhau — KHÔNG được lẫn** (`run.go:55-90`):
   - **Tạo mới** = `--headless --prompt-file prompt.txt`. Nhánh `prompt != ""` gọi `PrepareQuick` → `StartPrepared` = **dựng truyện từ đầu**.
   - **Resume** = `--headless` **KHÔNG truyền prompt**. Chỉ khi prompt rỗng mới vào nhánh `eng.Resume()` khôi phục từ checkpoint.
   - **Nguy hiểm:** nếu resume mà vẫn gửi `--prompt-file`, CLI sẽ **architecting lại từ đầu, đè lên truyện đang có** → mất nội dung. Orchestrator PHẢI dùng đúng dạng lệnh theo trạng thái.
   - **Biên:** nếu truyện chết quá sớm (chưa resumable — `IsResumable` = `phase==writing && current_chapter>0`, `runtime.go:65`), resume-không-prompt sẽ lỗi `"yêu cầu --prompt hoặc thư mục phải có phiên khôi phục"` (`run.go:84`). → Orchestrator phải bắt lỗi này và fallback: hoặc báo "không thể resume, cần tạo lại", hoặc (nếu output rỗng) cho phép chạy lại với prompt.

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
   │  │  - Serve React/Vite static dist         │  │
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
1. Sinh `slug` từ tên truyện (unique, an toàn filesystem — mục 10.2).
2. Tạo `data/novels/<slug>/`, copy/symlink `config.json` (có thể chèn `style` theo lựa chọn).
3. Ghi prompt ra `data/novels/<slug>/prompt.txt`.
4. `exec.Command(bin, "--config", cfg, "--headless", "--prompt-file", "prompt.txt")` với `cmd.Dir = data/novels/<slug>/`.
5. Lưu record vào SQLite (slug, tên, pid, trạng thái, thời điểm).
6. Đăng ký watcher lên **thư mục** `output/novel/meta/` (mục 7).

**Resume truyện đã dừng/lỗi** = orchestrator (KHÁC lệnh tạo mới — mục 1.9):
1. `exec.Command(bin, "--config", cfg, "--headless")` — **TUYỆT ĐỐI không `--prompt`/`--prompt-file`** — với `cmd.Dir = data/novels/<slug>/`.
2. Nếu process thoát ngay với lỗi "yêu cầu --prompt hoặc thư mục phải có phiên khôi phục" → truyện chưa từng vào writing → đánh dấu `error`, không tự chạy lại với prompt (tránh đè), để người dùng quyết định.
3. Cập nhật pid + status `running`, gắn lại watcher.

> **Bất biến (invariant) phải giữ:** không bao giờ truyền prompt vào một `data/novels/<slug>/` đã có `output/`. Một slug = một truyện trọn đời.

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
| `POST` | `/api/novels/{slug}/stop` | Dừng process (**hard kill** — CLI không bắt signal, mục 1.8; checkpoint atomic giữ chương đã xong, mất chương dở) |
| `POST` | `/api/novels/{slug}/resume` | Spawn lại **KHÔNG prompt** (mục 1.9, mục 4) → CLI tự resume từ checkpoint |
| `GET`  | `/api/novels/{slug}/events` | **SSE stream**: progress, log tail, sự kiện chương mới |
| `GET`  | `/api/novels/{slug}/export?fmt=txt\|epub` | Gọi trực tiếp `internal/host/exp.Run` trong web process; không spawn CLI `/export` vì `/export` là lệnh TUI |
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

> **Lưu ý nền (mục 1.7):** output ghi atomic qua rename → không lo đọc rách, nhưng **phải watch THƯ MỤC chứ không watch file**, vì rename thay inode sẽ làm watch-trên-file chết sau lần ghi đầu.

1. **fsnotify** add watch lên **thư mục** `data/novels/<slug>/output/novel/meta/` và thư mục `chapters/` (KHÔNG add trực tiếp lên `progress.json`).
2. Lọc event theo tên: chỉ phản ứng khi `meta/progress.json` xuất hiện qua event `CREATE`/`RENAME` (atomic rename bắn 2 event này, không phải `WRITE`).
3. **Debounce 200ms**: gom nhiều event sát nhau, đọc `progress.json` 1 lần ở rìa sau → so sánh field → đẩy event SSE + cập nhật SQLite. (Atomic rename loại bỏ torn-read, debounce chỉ để giảm nhiễu.)
4. Khi có file `chapters/NN.md` mới → bắn `chapter_done`.
5. **stdout/stderr** của process được pipe vào `logs/<slug>.log` + tail 50 dòng cuối đẩy qua SSE (`type:"log"`).
6. Frontend React dùng `EventSource` để cập nhật thanh tiến độ/log/chương mới không cần reload.

> Phương án dự phòng nếu fsnotify lỗi trên 1 số filesystem (vd volume Docker/mạng): **poll** `progress.json` mỗi 2–3s. Rẻ vì file nhỏ. Quyết định: bật cả hai — fsnotify là chính, một ticker poll 5s làm lưới an toàn (reconcile nếu lỡ event).

---

## 8. Dừng/Khôi phục & Can thiệp real-time

### 8.0. Ngữ nghĩa "stop" (đã xác minh — mục 1.8)
- CLI **không bắt signal nào** → "stop" thực chất là kill process. Trên Linux dùng `cmd.Process.Kill()` (SIGKILL) hoặc `Signal(SIGTERM)`; cả hai đều **không** chạy defer cleanup của headless.
- **Vì sao vẫn an toàn:** mọi chương xong + checkpoint đã ghi atomic (mục 1.7) trước thời điểm kill. Resume (spawn lại cùng cwd) khôi phục từ checkpoint cuối → chỉ chương đang viết dở bị làm lại.
- **Hệ quả vận hành:** sau `stop`, đánh dấu status `stopped`; orchestrator phải `Wait()` để reap zombie + ghi exit code. Đừng hứa "lưu sạch tiến độ đang viết" trên UI — chỉ hứa "giữ tới chương đã hoàn thành gần nhất".

### 8.0b. Suy ra `status` từ (exit code + progress.phase)
Process kết thúc KHÔNG đồng nghĩa truyện xong. Orchestrator map như sau khi `Wait()` trả về:
- **Web chủ động kill** (do `/stop`) → `stopped` (bất kể exit code; orchestrator biết mình vừa kill).
- **Exit 0** + `progress.phase == "complete"` → `done` (lưu ý: `complete` là giá trị thật trong `internal/domain`; `done` chỉ là status DB/UI).
- **Exit 0** nhưng phase chưa xong → bất thường (CLI thoát sớm) → `error` + ghi log.
- **Exit ≠ 0 / bị kill bởi tín hiệu** (không do web) → `error` (vd rớt mạng LLM) → hiện nút **resume**.
→ Đọc `progress.phase` ngay sau khi process thoát, đừng chỉ tin exit code.

### 8.1. Can thiệp sống (Phase 3 — cần khảo sát thêm)
- TUI cho phép gõ can thiệp khi đang viết (README mục "Can thiệp thời gian thực"); cơ chế lưu qua `internal/store/directives.go` + `signals.go`.
- **Headless hiện là "fire-and-forget"** — không có kênh nhập liệu khi đang chạy.
- Hướng khả thi (chưa xác minh đủ, để Phase 3):
  - **8a.** Web ghi directive vào file mà coordinator đang chạy đọc được (`directives`/`signals` store) → cần kiểm tra coordinator có poll các file này trong vòng lặp headless không.
  - **8b.** Đơn giản hơn: web cho phép **stop → sửa directive → resume**. An toàn, không cần đụng CLI.
- **V1 chốt:** chỉ đọc + tạo + stop/resume. Can thiệp sống để sau.

---

## 9. Bảo mật & vận hành VPS

- **Auth:** tối thiểu 1 lớp — chọn **một** trong 2 đường:
  - **Đơn giản nhất:** Basic Auth ở Caddy/Nginx, app chỉ nghe `127.0.0.1:8080`.
  - **Nếu auth trong app:** single password + session cookie; mật khẩu lưu dạng hash (`bcrypt`/`argon2id`), cookie bắt buộc `HttpOnly`, `Secure`, `SameSite=Lax/Strict`; các POST/DELETE có CSRF token; login có rate limit/backoff.
- **TLS:** đặt sau **Caddy** (tự Let's Encrypt) hoặc **Nginx**. `ainovel-web` chỉ nghe `127.0.0.1:8080`, proxy lo HTTPS.
- **API key LLM:** chỉ nằm trong `config/config.json` trên VPS (chmod 600), không lộ ra frontend.
- **Giới hạn tài nguyên:** 2 core/4GB → nên giới hạn **số process song song** (vd tối đa 2–3 truyện viết cùng lúc) bằng một **semaphore/queue** trong orchestrator. Truyện thứ N+1 ở trạng thái `queued`.
- **systemd** quản lý `ainovel-web` (auto-restart). Process CLI là con của web; web chịu trách nhiệm reap + ghi exit code.
- **Sao lưu:** `data/` là toàn bộ tài sản → cron `tar`/rsync định kỳ.

### 9.1. Startup recovery khi `ainovel-web` restart
Khi web server khởi động lại, không được tin tuyệt đối `pid/status` cũ trong SQLite:
1. Mở SQLite, scan `data/novels/*` và reconcile record còn thiếu.
2. Với record `running/queued`: kiểm tra process cũ còn sống hay không (pid có thể tái sử dụng, nên nếu không chắc thì coi là không tin cậy).
3. Nếu DB ghi `running` nhưng process không còn do web/systemd restart → đọc `progress.json`; nếu `phase=="complete"` thì mark `done`, ngược lại mark `error` hoặc `stopped` kèm `last_error="web restarted; process not attached"`.
4. Gắn lại watcher/poll cho mọi truyện chưa `done` để UI vẫn cập nhật file mới nếu có process ngoài còn ghi.
5. **Không tự resume** khi boot, trừ khi sau này thêm tùy chọn explicit. Resume tự động dễ nhân đôi process hoặc resume nhầm trạng thái.

---

## 10. Cấu trúc thư mục code (đề xuất)

Đặt server trong cùng repo để tái dùng `internal/domain`:

```
cmd/ainovel-web/main.go            # entrypoint server
internal/web/
├── server.go                      # router, middleware, mount API/SSE/static dist
├── api_novels.go                  # handlers REST
├── sse.go                         # hub SSE + broadcast
├── orchestrator.go                # sở hữu map[slug]*NovelProc, semaphore/queue (mục 10.1)
├── proc.go                        # NovelProc: 1 process + watcher + log tail, vòng đời riêng
├── watcher.go                     # fsnotify (watch THƯ MỤC, mục 7) → events
├── indexdb.go                     # SQLite (modernc.org/sqlite)
├── reader.go                      # đọc & parse file output (dùng internal/domain)
├── slug.go                        # sinh slug unique, an toàn filesystem (mục 10.2)
├── render.go                      # markdown→HTML + sanitize (mục 10.3)
└── auth.go                        # session/password
web/frontend/
├── package.json                   # React + Vite + TypeScript
├── index.html
├── vite.config.ts                 # dev proxy /api + /events về Go server
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   ├── api/                       # fetch client + EventSource helpers
│   ├── components/                # dashboard, progress, tabs, log tail
│   ├── pages/                     # list/detail/chapter/create
│   ├── types/                     # DTO khớp REST API
│   └── styles/                    # CSS/Tailwind entry
└── dist/                          # build output, Go serve ở production
```

### 10.0. Frontend React/Vite (SPA)
Frontend là một SPA build-time, backend Go là API + static file server:
- Dev: chạy Vite dev server, proxy `/api/*` và `/api/novels/*/events` sang Go backend.
- Prod: `npm run build` tạo `web/frontend/dist`; `ainovel-web` serve `dist/index.html` cho route không phải `/api`.
- Realtime: dùng `EventSource` trực tiếp; mỗi trang detail mở 1 stream `/api/novels/{slug}/events`, đóng stream khi rời trang.
- API types: định nghĩa DTO TypeScript ở `src/types`; giữ tên field JSON theo backend, không tự suy diễn schema từ file output.
- UI library: ưu tiên CSS nhẹ + `lucide-react` icons. Có thể thêm thư viện chuyên biệt khi cần: `react-markdown`, `recharts`, `cytoscape`/`reactflow`, `tanstack-table`.
- Không render HTML chưa sanitize từ client. Nội dung chương nên nhận từ backend dưới dạng HTML đã sanitize hoặc markdown escaped rồi render bằng component an toàn.

### 10.1. Concurrency ownership (rủi ro #1 — phải chốt trước khi code Phase 2)
Có N process + N watcher + SSE hub + DB sync chạy song song → đây là chỗ dễ race nhất. Quy ước sở hữu state:
- **`Orchestrator`** giữ `mu sync.Mutex` + `procs map[string]*NovelProc` + `sem chan struct{}` (semaphore = 2). Mọi create/stop/resume/list đi qua method của nó; không ai sờ map trực tiếp.
- **`NovelProc`** sở hữu trọn vòng đời 1 truyện: `*exec.Cmd`, watcher, log tailer, và 1 channel `events`. Chỉ goroutine của chính nó ghi vào file SQLite-row của nó → không 2 goroutine cùng update 1 slug.
- **SSE hub** chỉ *nhận* event từ các `NovelProc.events` (fan-in qua 1 channel) rồi *fan-out* tới subscriber. Hub không gọi ngược orchestrator.
- Reap: mỗi `NovelProc` tự `cmd.Wait()` trong goroutine riêng → khi xong, gửi event `status` cuối + nhả semaphore + tự gỡ khỏi map (qua orchestrator method có lock).
- Quy tắc: **không khóa lồng nhau** (đừng giữ orchestrator.mu khi gọi vào proc). Truyền dữ liệu qua channel, không qua shared pointer có lock.

### 10.1b. Queue scheduler (truyện thứ 3+)
Semaphore = 2 chỉ giới hạn process đang chạy; cần thêm scheduler rõ ràng cho trạng thái `queued`:
1. `CreateNovel` ghi record `queued` trước. Nếu còn slot semaphore thì đổi sang `running` và spawn ngay; nếu hết slot thì để nguyên `queued`.
2. Khi `NovelProc.Wait()` kết thúc, proc gọi orchestrator method `onProcDone(slug, result)` để nhả slot, cập nhật status, gỡ map.
3. Sau khi nhả slot, orchestrator gọi `tryStartQueuedLocked()` để lấy truyện `queued` cũ nhất (`created_at ASC`) và spawn.
4. `resume` cũng đi qua cùng semaphore: nếu hết slot thì đặt `queued` với `last_error` giữ nguyên, không spawn vượt giới hạn.
5. Không giữ `orchestrator.mu` trong lúc `cmd.Start()`/khởi tạo watcher/log tailer; chuẩn bị state dưới lock, thả lock rồi spawn, sau đó commit trạng thái qua method có lock.

### 10.2. slug.go — an toàn cho tên tiếng Việt
Tên truyện có dấu + khoảng trắng. Pipeline: NFD normalize → strip dấu (`unicode.Mn`) → lowercase → thay non-`[a-z0-9]` bằng `-` → trim `-` → chống trùng (thêm đuôi `-2`,`-3` nếu slug đã tồn tại trong SQLite). Bắt buộc vì slug = tên thư mục trên đĩa.

### 10.3. render.go — sanitize output
Chương do LLM sinh → render `goldmark` rồi **bắt buộc qua `bluemonday.UGCPolicy()`** trước khi trả HTML cho frontend, tránh XSS nếu model phun thẻ HTML/script. (Plan gốc lo API key nhưng quên output escaping.)

### 10.4. export.go — TXT/EPUB qua web
Web export dùng trực tiếp package nội bộ, không gọi CLI:
- Tạo `store.NewStore(data/novels/<slug>/output/novel)`.
- Gọi `internal/host/exp.Run(ctx, exp.Deps{Store: store}, exp.Options{Format: ..., From: ..., To: ..., Overwrite: true})`.
- Output mặc định nên ghi vào `data/novels/<slug>/exports/` hoặc temp file download, không ghi lung tung theo input người dùng.
- Nếu cho chọn `OutPath`, phải resolve path nằm trong vùng cho phép (`exports/`) để tránh path traversal.
- Endpoint trả file download và metadata số chương/bytes/skipped.

### 10.5. delete.go — xóa truyện an toàn
`DELETE /api/novels/{slug}` là thao tác nguy hiểm, phải làm chặt:
- Không cho xóa nếu truyện đang `running`; bắt người dùng stop trước.
- Slug từ URL phải qua validator `[a-z0-9-]+`, rồi resolve absolute path và kiểm tra chắc chắn nằm dưới `data/novels`.
- Xác nhận 2 lớp ở UI: nhập lại slug hoặc tên truyện.
- V1 nên **move sang `data/trash/<slug>-<timestamp>/`** rồi cập nhật DB `deleted_at`/status `deleted` thay vì xóa vĩnh viễn ngay.
- Chỉ có job cleanup thủ công/định kỳ mới xóa hẳn trash sau khi backup.

**Phụ thuộc mới (tối thiểu):**
- `github.com/fsnotify/fsnotify` — watch thư mục.
- `modernc.org/sqlite` — SQLite pure-Go (không CGO, build tĩnh dễ).
- `github.com/yuin/goldmark` — markdown→HTML (kiểm tra go.mod xem đã có chưa; `internal/host/exp` có thể đã dùng).
- `github.com/microcosm-cc/bluemonday` — sanitize HTML output.
- Frontend npm: `react`, `react-dom`, `vite`, `typescript`, `lucide-react`.
- Frontend tùy chọn theo Phase 3: `react-markdown`, `recharts`, `cytoscape`/`reactflow`, `@tanstack/react-table`.

---

## 11. Lộ trình thực thi (phân giai đoạn)

### Phase 0 — Móng (đọc dữ liệu)
- [ ] `cmd/ainovel-web` + router cơ bản, phục vụ REST API + static `web/frontend/dist`.
- [ ] `web/frontend` React + Vite + TypeScript skeleton, dev proxy `/api` về Go server.
- [ ] `reader.go`: parse `progress.json`, `premise.md`, `outline`, `characters` từ 1 working dir có sẵn (dùng chính `output/novel` hiện tại để test).
- [ ] React trang **list** + trang **chi tiết** (đọc tĩnh qua REST API, chưa real-time).
- _Done khi:_ mở web thấy được truyện "Người Vẽ Hồn" đang có trong `output/`.

### Phase 1 — Đa truyện + tạo mới
- [ ] SQLite index + đồng bộ từ `progress.json`.
- [ ] `orchestrator.go`: tạo working dir, spawn `--headless --prompt-file`, lưu pid.
- [ ] Queue scheduler: semaphore = 2, truyện dư ở `queued`, tự chạy truyện queued cũ nhất khi slot trống.
- [ ] React form "Tạo truyện mới" (title, prompt, chọn style/model) gọi `POST /api/novels`.
- [ ] Startup recovery: reconcile DB + `data/novels/*`, không tin pid cũ sau restart.
- _Done khi:_ bấm nút tạo → truyện mới bắt đầu viết, hiện trong danh sách.

### Phase 2 — Real-time
- [ ] `watcher.go` (fsnotify) + `sse.go` hub.
- [ ] React `EventSource` client: thanh tiến độ live, "chương mới xong" tự hiện, tail log.
- [ ] Stop (**hard kill + Wait/reap**) / Resume (spawn lại **không prompt**).
- _Done khi:_ ngồi máy khác xem chương nhảy số real-time không cần F5.

### Phase 3 — Nâng cao
- [ ] Đọc chương (markdown render), tab nhân vật/thế giới/phục bút/timeline/review.
- [ ] UI nâng cao nếu cần: bảng bằng TanStack Table, chart bằng Recharts, graph quan hệ bằng Cytoscape/React Flow.
- [ ] Export TXT/EPUB qua web bằng `internal/host/exp.Run`.
- [ ] Xóa truyện an toàn: không xóa khi running, validate slug/path, move vào trash trước.
- [ ] (Khảo sát) Can thiệp real-time — mục 8.
- [ ] Auth + TLS + systemd + backup.

### 11.1. Checklist test invariant trước khi triển khai VPS
- [ ] Create truyện mới luôn spawn với `--headless --prompt-file prompt.txt` trong dir mới, có `cmd.Dir=<slug>`.
- [ ] Resume luôn spawn `--headless` **không prompt** trong dir cũ.
- [ ] Không bao giờ truyền prompt vào slug đã có `output/`.
- [ ] `Wait()` map đúng `progress.phase=="complete"` → DB status `done`.
- [ ] Stop từ web mark `stopped`, gọi `Wait()` để reap, và resume viết lại từ checkpoint gần nhất.
- [ ] Watcher nhận nhiều lần atomic rename của `meta/progress.json` vì watch thư mục, không watch file.
- [ ] Poll 5s reconcile được nếu fsnotify miss event.
- [ ] Queue chỉ chạy tối đa 2 process, truyện queued cũ nhất tự chạy khi slot trống.
- [ ] Restart `ainovel-web` không tự resume và không tin pid cũ.
- [ ] Markdown chương render qua sanitize trước khi đưa ra frontend/DOM.
- [ ] Export chỉ ghi/đọc trong vùng cho phép.
- [ ] Delete chỉ move vào trash sau xác nhận, không xóa khi process đang chạy.

---

## 12. Rủi ro & lưu ý

| Rủi ro | Giảm thiểu |
|---|---|
| Genre mới không xuất hiện vì assets nhúng tĩnh | Rebuild CLI binary/Docker **sau khi** thêm genre; CI build kèm `assets/` |
| `OutputDir` không đổi được qua config | Dùng **working dir riêng/truyện** (đã chọn). _Tùy chọn:_ patch CLI thêm `--output-dir`/env nếu muốn gọn hơn |
| fsnotify watch-trên-file chết sau rename (mục 1.7) | **Watch thư mục** `meta/`, lọc theo tên + lắng nghe `CREATE`/`RENAME`, không `WRITE` |
| fsnotify không bắn trên volume Docker/mạng | Fallback **poll** progress.json 5s (lưới an toàn, chạy song song fsnotify) |
| Race giữa N process/watcher/SSE/DB | **Ownership rõ ràng (mục 10.1)**: Orchestrator giữ map có lock, mỗi NovelProc sở hữu row riêng, fan-in qua channel, không khóa lồng |
| Queue không tự chạy tiếp sau khi slot trống | `onProcDone` luôn gọi `tryStartQueuedLocked()` (mục 10.1b); test thứ tự `created_at ASC` |
| Web restart làm DB ghi `running` sai thực tế | Startup recovery reconcile DB + filesystem, không tin pid cũ, không auto resume (mục 9.1) |
| 2 process viết cùng lúc làm nghẽn 4GB | Process CLI nhẹ khi dùng cloud LLM, nhưng vẫn **giới hạn song song = 2** cho an toàn |
| "stop" làm mất tiến độ | CLI không bắt signal (mục 1.8) nhưng checkpoint atomic → chỉ mất chương dở. UI hứa đúng: "giữ tới chương xong gần nhất" |
| **Resume gửi nhầm prompt → architecting đè, mất truyện** | **Bất biến (mục 1.9, 4):** resume spawn KHÔNG prompt; tạo mới mới có prompt. Không bao giờ truyền prompt vào dir đã có `output/` |
| Headless chết giữa chừng (mất mạng LLM) | CLI có checkpoint → orchestrator `Wait()` thấy exit≠0 → đánh dấu `error`, cho **resume** 1 nút |
| Exit 0 nhưng truyện chưa xong | Sau `Wait()`, chỉ map `done` khi `progress.phase=="complete"`; phase khác là `error` (mục 8.0b) |
| Lệch schema khi CLI nâng cấp | Parse bằng `internal/domain` thay vì struct tự chế; bọc try-parse, degrade graceful |
| Frontend React lệch DTO backend | Định nghĩa DTO TypeScript rõ trong `web/frontend/src/types`, contract test/API smoke test cho endpoint chính |
| Vite build làm deploy phức tạp hơn HTMX | Chỉ cần Node ở máy build/CI; production Go serve `web/frontend/dist`, không chạy Node trên VPS |
| XSS từ output LLM | `bluemonday.UGCPolicy()` sau goldmark (mục 10.3) |
| Path traversal khi export/delete | Validate slug, resolve absolute path nằm dưới thư mục cho phép; export chỉ trong `exports/`, delete move vào `trash/` (mục 10.4–10.5) |
| Slug đụng tên tiếng Việt/trùng | NFD strip dấu + chống trùng đuôi số (mục 10.2) |
| Lộ API key | Key chỉ ở `config.json` (chmod 600), không gửi xuống frontend |

---

## 13. Quyết định đã chốt (sẵn sàng code)

| # | Vấn đề | Quyết định |
|---|---|---|
| 1 | Sửa CLI? | **Không sửa ở V1.** Dùng working dir riêng/truyện. Cân nhắc patch `--output-dir` (~10 dòng) ở Phase 1 nếu thấy layout cwd phiền — chỉ khi bạn xác nhận. |
| 2 | Auth | **Single password + session cookie** hoặc **Basic Auth ở reverse proxy**. Nếu auth trong app: hash mật khẩu, cookie an toàn, CSRF cho POST/DELETE. Không multi-user. |
| 3 | Song song | **2** process viết cùng lúc (semaphore), truyện thứ 3+ vào `queued`. |
| 4 | UI | **React + Vite + TypeScript SPA**, Go serve static `dist` ở production. |
| 5 | Triển khai | **Binary trần + systemd + static dist**. Genre nhúng `go:embed` nên Docker không thêm lợi ích, chỉ phiền rebuild. |

> Tất cả giả định cốt lõi đã xác minh khỏi source (mục 1.1–1.8). **Sẵn sàng bắt đầu Phase 0.**

### 13.1. Câu hỏi còn lại cho bạn (không chặn Phase 0, cần trước Phase 1)
- **Genre nào cho form "tạo truyện mới"?** Hiện có `kinh_di_vietnam`. Liệt kê cứng từ `assets/` hay quét thư mục lúc build?
- **Provider/model mặc định** trên VPS: DeepSeek hay OpenRouter? (ảnh hưởng `config.json` gốc.)
- **Domain + TLS**: đã có domain trỏ về VPS chưa, hay dùng IP + self-signed tạm?
