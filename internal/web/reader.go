package web

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

// NovelDir là thư mục output/novel của một truyện.
type NovelDir struct {
	Slug string
	Dir  string // absolute path tới output/novel
}

// NovelSummary là DTO danh sách truyện.
type NovelSummary struct {
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	Phase          string `json:"phase"`
	CurrentChapter int    `json:"current_chapter"`
	TotalChapters  int    `json:"total_chapters"`
	CompletedCount int    `json:"completed_count"`
	WordCount      int    `json:"word_count"`
	Style          string `json:"style,omitempty"`
	Model          string `json:"model,omitempty"`
	UpdatedAt      int64  `json:"updated_at"`
}

// NovelDetail gộp progress + run meta.
type NovelDetail struct {
	NovelSummary
	Progress *domain.Progress `json:"progress"`
	RunMeta  *domain.RunMeta  `json:"run_meta,omitempty"`
}

// PremiseResponse nội dung premise.md.
type PremiseResponse struct {
	Content string `json:"content"`
}

// OutlineResponse đề cương phẳng hoặc phân tầng.
type OutlineResponse struct {
	Layered bool                    `json:"layered"`
	Entries []domain.OutlineEntry   `json:"entries,omitempty"`
	Volumes []domain.VolumeOutline  `json:"volumes,omitempty"`
}

// Reader đọc dữ liệu truyện từ thư mục output/novel.
type Reader struct {
	dataRoot string
	bySlug   map[string]NovelDir
}

// NewReader quét dataRoot và lập catalog slug → novel dir.
func NewReader(dataRoot string) (*Reader, error) {
	entries, err := discoverNovels(dataRoot)
	if err != nil {
		return nil, err
	}
	bySlug := make(map[string]NovelDir, len(entries))
	for _, e := range entries {
		bySlug[e.Slug] = e
	}
	return &Reader{dataRoot: dataRoot, bySlug: bySlug}, nil
}

// ListNovels trả về tất cả truyện đã discover.
func (r *Reader) ListNovels() ([]NovelSummary, error) {
	out := make([]NovelSummary, 0, len(r.bySlug))
	for slug := range r.bySlug {
		summary, err := r.Summary(slug)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	sortSummaries(out)
	return out, nil
}

// Summary đọc metadata tóm tắt một truyện.
func (r *Reader) Summary(slug string) (NovelSummary, error) {
	nd, ok := r.bySlug[slug]
	if !ok {
		return NovelSummary{}, errNotFound(slug)
	}
	st := store.NewStore(nd.Dir)
	progress, err := st.Progress.Load()
	if err != nil {
		return NovelSummary{}, fmt.Errorf("progress %s: %w", slug, err)
	}
	if progress == nil {
		return NovelSummary{}, fmt.Errorf("progress %s: empty", slug)
	}
	runMeta, _ := st.RunMeta.Load()
	title := progress.NovelName
	if title == "" {
		title = slug
	}
	updatedAt := fileMtime(filepath.Join(nd.Dir, "meta", "progress.json"))
	return NovelSummary{
		Slug:           slug,
		Title:          title,
		Status:         statusFromPhase(progress.Phase),
		Phase:          string(progress.Phase),
		CurrentChapter: progress.CurrentChapter,
		TotalChapters:  progress.TotalChapters,
		CompletedCount: len(progress.CompletedChapters),
		WordCount:      progress.TotalWordCount,
		Style:          runMetaStyle(runMeta),
		Model:          runMetaModel(runMeta),
		UpdatedAt:      updatedAt,
	}, nil
}

// Detail đọc chi tiết truyện.
func (r *Reader) Detail(slug string) (NovelDetail, error) {
	summary, err := r.Summary(slug)
	if err != nil {
		return NovelDetail{}, err
	}
	nd := r.bySlug[slug]
	st := store.NewStore(nd.Dir)
	progress, err := st.Progress.Load()
	if err != nil {
		return NovelDetail{}, err
	}
	runMeta, _ := st.RunMeta.Load()
	return NovelDetail{
		NovelSummary: summary,
		Progress:     progress,
		RunMeta:      runMeta,
	}, nil
}

// Premise đọc premise.md.
func (r *Reader) Premise(slug string) (PremiseResponse, error) {
	nd, ok := r.bySlug[slug]
	if !ok {
		return PremiseResponse{}, errNotFound(slug)
	}
	st := store.NewStore(nd.Dir)
	content, err := st.Outline.LoadPremise()
	if err != nil {
		return PremiseResponse{}, err
	}
	return PremiseResponse{Content: content}, nil
}

// Outline đọc outline.json hoặc layered_outline.json.
func (r *Reader) Outline(slug string) (OutlineResponse, error) {
	nd, ok := r.bySlug[slug]
	if !ok {
		return OutlineResponse{}, errNotFound(slug)
	}
	st := store.NewStore(nd.Dir)
	volumes, err := st.Outline.LoadLayeredOutline()
	if err != nil {
		return OutlineResponse{}, err
	}
	if len(volumes) > 0 {
		return OutlineResponse{Layered: true, Volumes: volumes}, nil
	}
	entries, err := st.Outline.LoadOutline()
	if err != nil {
		return OutlineResponse{}, err
	}
	return OutlineResponse{Layered: false, Entries: entries}, nil
}

// Characters đọc characters.json.
func (r *Reader) Characters(slug string) ([]domain.Character, error) {
	nd, ok := r.bySlug[slug]
	if !ok {
		return nil, errNotFound(slug)
	}
	st := store.NewStore(nd.Dir)
	return st.Characters.Load()
}

func statusFromPhase(phase domain.Phase) string {
	if phase == domain.PhaseComplete {
		return "done"
	}
	return "idle"
}

func runMetaStyle(m *domain.RunMeta) string {
	if m == nil {
		return ""
	}
	return m.Style
}

func runMetaModel(m *domain.RunMeta) string {
	if m == nil {
		return ""
	}
	return m.Model
}

func fileMtime(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().Unix()
}

type notFoundError struct{ slug string }

func (e notFoundError) Error() string { return "novel not found: " + e.slug }

func errNotFound(slug string) error { return notFoundError{slug: slug} }

func isNotFound(err error) bool {
	_, ok := err.(notFoundError)
	return ok
}

// discoverNovels tìm mọi output/novel có meta/progress.json.
func discoverNovels(dataRoot string) ([]NovelDir, error) {
	var found []NovelDir
	used := make(map[string]bool)

	add := func(dir string, preferredSlug string) error {
		progressPath := filepath.Join(dir, "meta", "progress.json")
		if _, err := os.Stat(progressPath); err != nil {
			return nil
		}
		st := store.NewStore(dir)
		progress, err := st.Progress.Load()
		if err != nil || progress == nil {
			return nil
		}
		slug := preferredSlug
		if slug == "" {
			slug = uniqueSlug(progress.NovelName, used)
		} else {
			used[slug] = true
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		found = append(found, NovelDir{Slug: slug, Dir: abs})
		return nil
	}

	// Layout dev: output/novel
	legacy := filepath.Join(dataRoot, "output", "novel")
	if err := add(legacy, ""); err != nil {
		return nil, err
	}

	// Layout VPS: data/novels/<slug>/output/novel
	novelsRoot := filepath.Join(dataRoot, "data", "novels")
	entries, err := os.ReadDir(novelsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return found, nil
		}
		return nil, err
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := filepath.Join(novelsRoot, ent.Name(), "output", "novel")
		if err := add(dir, ent.Name()); err != nil {
			return nil, err
		}
	}
	return found, nil
}

func sortSummaries(items []NovelSummary) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].UpdatedAt > items[j-1].UpdatedAt; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// ReloadCatalog quét lại filesystem (Phase 0: gọi khi khởi động).
func (r *Reader) ReloadCatalog() error {
	entries, err := discoverNovels(r.dataRoot)
	if err != nil {
		return err
	}
	bySlug := make(map[string]NovelDir, len(entries))
	for _, e := range entries {
		bySlug[e.Slug] = e
	}
	r.bySlug = bySlug
	return nil
}

// SlugExists kiểm tra slug có trong catalog.
func (r *Reader) SlugExists(slug string) bool {
	_, ok := r.bySlug[slug]
	return ok
}

// ProgressMtime trả về thời điểm sửa progress.json (test helper).
func (r *Reader) ProgressMtime(slug string) time.Time {
	nd, ok := r.bySlug[slug]
	if !ok {
		return time.Time{}
	}
	info, err := os.Stat(filepath.Join(nd.Dir, "meta", "progress.json"))
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}