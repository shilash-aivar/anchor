package dashboard

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"anchor/internal/awsx"
	"anchor/internal/config"
	"anchor/internal/kubecfg"
	"anchor/internal/kube"
	"anchor/internal/session"
)

//go:embed static/*
var staticFiles embed.FS

type Options struct {
	Addr    string
	Version string
	Open    bool
}

func Run(opts Options) error {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:8765"
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}

	s := &Server{version: opts.Version}
	mux := http.NewServeMux()

	static, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/api/overview", s.handleOverview)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/use", s.handleUse)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/doctor", s.handleDoctor)
	mux.HandleFunc("/api/recent", s.handleRecent)
	mux.HandleFunc("/api/lint", s.handleLint)
	mux.HandleFunc("/api/prune", s.handlePrune)
	mux.HandleFunc("/api/commands", s.handleCommands)
	mux.HandleFunc("/api/namespaces", s.handleNamespaces)
	mux.HandleFunc("/api/ns", s.handleNS)
	mux.HandleFunc("/api/share", s.handleShare)
	mux.HandleFunc("/api/audit", s.handleAudit)
	mux.HandleFunc("/api/find", s.handleFind)
	mux.HandleFunc("/api/open-link", s.handleOpenLink)
	mux.HandleFunc("/api/pods", s.handlePods)

	srv := &http.Server{Addr: opts.Addr, Handler: cors(mux), ReadHeaderTimeout: 5 * time.Second}
	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String()
	fmt.Printf("anchor dashboard → %s\n", url)
	fmt.Println("Press Ctrl+C to stop")
	if opts.Open {
		openBrowser(url)
	}
	return srv.Serve(ln)
}

type Server struct {
	version string
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	o, err := buildOverview(s.version)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sv, err := buildSessionView()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sv == nil {
		writeJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	writeJSON(w, http.StatusOK, sv)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projects, err := loadProjectSummaries()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

type useRequest struct {
	Project     string `json:"project"`
	Namespace   string `json:"namespace"`
	SkipConfirm bool   `json:"skip_confirm"`
	ConfirmText string `json:"confirm_text"`
}

func (s *Server) handleUse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req useRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.Project == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project required"})
		return
	}
	result, err := activateProject(req.Project, req.Namespace, req.ConfirmText, req.SkipConfirm)
	if err != nil {
		var cre *ConfirmRequiredError
		if errors.As(err, &cre) {
			writeJSON(w, http.StatusPreconditionRequired, map[string]any{
				"error":        "confirm_required",
				"confirm_text": cre.Text,
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	sv, _ := buildSessionView()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": sv, "message": fmt.Sprintf("Activated %s / %s", result.State.Project, result.State.Namespace)})
}

type loginRequest struct {
	Profile string `json:"profile"`
	All     bool   `json:"all"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginRequest
	_ = readJSON(r, &req)
	if r.URL.Query().Get("all") == "true" {
		req.All = true
	}
	if req.All {
		projects, err := config.LoadAllProjects()
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		seen := map[string]bool{}
		for _, p := range projects {
			if p.AWSProfile == "" || seen[p.AWSProfile] {
				continue
			}
			seen[p.AWSProfile] = true
			if err := awsx.SSOLogin(p.AWSProfile); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "logged in all profiles"})
		return
	}
	profile := req.Profile
	if profile == "" {
		if sv, _ := buildSessionView(); sv != nil {
			profile = sv.AWSProfile
		}
	}
	if err := awsx.SSOLogin(profile); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "profile": profile})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	skip := r.URL.Query().Get("yes") == "true"
	var body struct {
		SkipConfirm bool `json:"skip_confirm"`
	}
	_ = readJSON(r, &body)
	if body.SkipConfirm {
		skip = true
	}
	writeJSON(w, http.StatusOK, syncAll(skip))
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, runDoctor())
}

func (s *Server) handleRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	recent, err := session.LoadRecent()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, recent)
}

func (s *Server) handleLint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	issues, err := kubecfg.LintAll()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, issues)
}

func (s *Server) handlePrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	dry := r.URL.Query().Get("dry_run") == "true"
	removed, err := kubecfg.PruneOrphans(dry)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"removed": removed, "dry_run": dry})
}

func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, AllCommands())
}

func (s *Server) handleNamespaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st, err := session.Load()
	if err != nil || st == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no active session"})
		return
	}
	ns, err := kube.ListNamespaces(st.Kubeconfig, st.KubeContext)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ns)
}

type nsRequest struct {
	Namespace string `json:"namespace"`
}

func (s *Server) handleNS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req nsRequest
	if err := readJSON(r, &req); err != nil || req.Namespace == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "namespace required"})
		return
	}
	st, err := switchNamespace(req.Namespace)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "namespace": st.Namespace})
}

func (s *Server) handleShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sv, err := buildSessionView()
	if err != nil || sv == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no active session"})
		return
	}
	block := map[string]string{
		"project": sv.Project, "tier": sv.Tier, "namespace": sv.Namespace,
		"context": sv.Context, "cluster": sv.Cluster, "region": sv.AWSRegion, "profile": sv.AWSProfile,
	}
	if sv.AccountID != "" {
		block["account_id"] = sv.AccountID
	}
	writeJSON(w, http.StatusOK, block)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	n := 50
	if v := r.URL.Query().Get("lines"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			n = i
		}
	}
	lines, err := tailAudit(n)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, lines)
}

func (s *Server) handleFind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "q required"})
		return
	}
	out, err := findResources(q)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": out})
}

type openLinkRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Server) handleOpenLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req openLinkRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	url := req.URL
	if url == "" && req.Name != "" {
		sv, err := buildSessionView()
		if err != nil || sv == nil || sv.Links == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "link not found"})
			return
		}
		var ok bool
		url, ok = sv.Links[req.Name]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown link"})
			return
		}
	}
	if url == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url or name required"})
		return
	}
	if err := openBrowser(url); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "url": url})
}

func (s *Server) handlePods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st, err := session.Load()
	if err != nil || st == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no active session"})
		return
	}
	pods, err := kube.ListPodLines(st.Kubeconfig, st.KubeContext, st.Namespace)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, pods)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch {
	case lookPath("open") == nil:
		cmd = exec.Command("open", url)
	case lookPath("xdg-open") == nil:
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("no browser opener available")
	}
	return cmd.Run()
}

func lookPath(name string) error {
	_, err := exec.LookPath(name)
	return err
}
