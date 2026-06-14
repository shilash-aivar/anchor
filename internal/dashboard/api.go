package dashboard

import (
	"anchor/internal/awsx"
	"anchor/internal/config"
	"anchor/internal/guard"
	"anchor/internal/kube"
	"anchor/internal/kubecfg"
	"anchor/internal/session"
	"anchor/internal/use"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type Overview struct {
	Version  string           `json:"version"`
	Session  *SessionView     `json:"session,omitempty"`
	Projects []ProjectSummary `json:"projects"`
	Recent   []session.RecentEntry `json:"recent,omitempty"`
	Commands []Command        `json:"commands"`
}

type SessionView struct {
	Project    string            `json:"project"`
	Tier       string            `json:"tier"`
	Namespace  string            `json:"namespace"`
	Context    string            `json:"context"`
	Cluster    string            `json:"cluster"`
	AWSProfile string            `json:"aws_profile"`
	AWSRegion  string            `json:"aws_region"`
	AccountID  string            `json:"account_id,omitempty"`
	Kubeconfig string            `json:"kubeconfig"`
	ReadOnly   bool              `json:"readonly"`
	UpdatedAt  string            `json:"updated_at"`
	AWSValid   bool              `json:"aws_valid"`
	AWSARN     string            `json:"aws_arn,omitempty"`
	AWSExpires string            `json:"aws_expires,omitempty"`
	AWSHint    string            `json:"aws_hint,omitempty"`
	ClusterOK  bool              `json:"cluster_ok"`
	ClusterErr string            `json:"cluster_err,omitempty"`
	Notes      string            `json:"notes,omitempty"`
	Links      map[string]string `json:"links,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

type ProjectSummary struct {
	Name             string            `json:"name"`
	Tier             string            `json:"tier"`
	Cluster          string            `json:"cluster"`
	AWSProfile       string            `json:"aws_profile"`
	Region           string            `json:"region"`
	DefaultNamespace string            `json:"default_namespace,omitempty"`
	ReadOnly         bool              `json:"readonly"`
	Active           bool              `json:"active"`
	Links            map[string]string `json:"links,omitempty"`
	NotesPreview     string            `json:"notes_preview,omitempty"`
}

type DoctorReport struct {
	Checks []DoctorCheck `json:"checks"`
}

type DoctorCheck struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Hint  string `json:"hint,omitempty"`
	Level string `json:"level,omitempty"`
}

type SyncResult struct {
	OK    int           `json:"ok"`
	Fail  int           `json:"fail"`
	Items []SyncProject `json:"items"`
}

type SyncProject struct {
	Project string `json:"project"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func buildSessionView() (*SessionView, error) {
	s, err := session.Load()
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	p, err := config.LoadProject(s.Project)
	if err != nil {
		return nil, err
	}
	v := &SessionView{
		Project:    s.Project,
		Tier:       s.Tier,
		Namespace:  s.Namespace,
		Context:    s.KubeContext,
		Cluster:    p.Cluster,
		AWSProfile: s.AWSProfile,
		AWSRegion:  s.AWSRegion,
		AccountID:  s.AccountID,
		Kubeconfig: s.Kubeconfig,
		ReadOnly:   p.ReadOnly,
		UpdatedAt:  s.UpdatedAt.Format(time.RFC3339),
		Notes:      p.Notes,
		Links:      p.Links,
		Env:        p.Env,
	}
	cred := awsx.CredentialStatusForProfile(s.AWSProfile)
	v.AWSValid = cred.Valid
	v.AWSExpires = cred.ExpiresIn
	v.AWSHint = cred.Hint
	if cred.Valid {
		if id, err := awsx.GetCallerIdentity(s.AWSProfile); err == nil {
			v.AWSARN = id.ARN
		}
	}
	if err := kube.ClusterReachable(s.Kubeconfig, s.KubeContext); err != nil {
		v.ClusterErr = err.Error()
	} else {
		v.ClusterOK = true
	}
	return v, nil
}

func buildOverview(version string) (*Overview, error) {
	projects, err := loadProjectSummaries()
	if err != nil {
		return nil, err
	}
	sv, err := buildSessionView()
	if err != nil {
		return nil, err
	}
	recent, _ := session.LoadRecent()
	return &Overview{
		Version:  version,
		Session:  sv,
		Projects: projects,
		Recent:   recent,
		Commands: AllCommands(),
	}, nil
}

func loadProjectSummaries() ([]ProjectSummary, error) {
	names, err := config.ListProjects()
	if err != nil {
		return nil, err
	}
	active := ""
	if s, _ := session.Load(); s != nil {
		active = s.Project
	}
	sort.Strings(names)
	out := make([]ProjectSummary, 0, len(names))
	for _, name := range names {
		p, err := config.LoadProject(name)
		if err != nil {
			continue
		}
		preview := strings.TrimSpace(p.Notes)
		if len(preview) > 120 {
			preview = preview[:117] + "..."
		}
		out = append(out, ProjectSummary{
			Name:             p.Name,
			Tier:             p.Tier,
			Cluster:          p.Cluster,
			AWSProfile:       p.AWSProfile,
			Region:           p.Region,
			DefaultNamespace: p.DefaultNamespace,
			ReadOnly:         p.ReadOnly,
			Active:           name == active,
			Links:            p.Links,
			NotesPreview:     preview,
		})
	}
	return out, nil
}

func runDoctor() DoctorReport {
	report := DoctorReport{}
	add := func(name string, ok bool, hint string) {
		report.Checks = append(report.Checks, DoctorCheck{Name: name, OK: ok, Hint: hint})
	}
	add("aws", awsx.AWSAvailable(), "install AWS CLI v2")
	add("kubectl", kube.KubectlAvailable(), "install kubectl")
	for _, t := range []struct{ n, h string }{
		{"fzf", "optional: brew install fzf"},
		{"stern", "brew install stern"},
		{"k9s", "brew install k9s"},
		{"helm", "brew install helm"},
	} {
		_, err := kube.LookPath(t.n)
		add(t.n, err == nil, t.h)
	}
	s, _ := session.Load()
	if s == nil {
		add("session", false, "no active project")
		return report
	}
	add("session", true, fmt.Sprintf("%s / %s / %s", s.Project, s.KubeContext, s.Namespace))
	cred := awsx.CredentialStatusForProfile(s.AWSProfile)
	if !cred.Valid {
		add("aws-auth", false, cred.Hint)
	} else if id, err := awsx.GetCallerIdentity(s.AWSProfile); err != nil {
		add("aws-auth", false, fmt.Sprintf("run anchor login %s", s.AWSProfile))
	} else {
		hint := id.Account
		if cred.ExpiresIn != "" {
			hint += " (" + cred.ExpiresIn + ")"
		}
		add("aws-auth", true, hint)
	}
	if err := kube.ClusterReachable(s.Kubeconfig, s.KubeContext); err != nil {
		add("cluster", false, err.Error())
	} else {
		add("cluster", true, "reachable")
	}
	issues, _ := kubecfg.LintAll()
	if len(issues) > 0 {
		add("lint", false, fmt.Sprintf("%d issue(s)", len(issues)))
	} else {
		add("lint", true, "")
	}
	return report
}

func activateProject(name, namespace, confirmText string, skip bool) (*use.Result, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	p, err := config.LoadProject(name)
	if err != nil {
		return nil, err
	}
	if !skip && p.ShouldConfirm(cfg.Options.ConfirmProduction) && confirmText == "" {
		return nil, &ConfirmRequiredError{Text: p.EffectiveConfirmText()}
	}
	if err := guard.ConfirmProjectSwitchTyped(p, cfg.Options.ConfirmProduction, skip, confirmText); err != nil {
		return nil, err
	}
	return use.ActivateSimple(name, namespace, true)
}

type ConfirmRequiredError struct {
	Text string
}

func (e *ConfirmRequiredError) Error() string {
	return "confirmation required"
}

func syncAll(skip bool) SyncResult {
	projects, err := config.LoadAllProjects()
	if err != nil {
		return SyncResult{Items: []SyncProject{{Project: "(all)", OK: false, Message: err.Error()}}}
	}
	res := SyncResult{}
	for _, p := range projects {
		item := SyncProject{Project: p.Name}
		r, err := use.PrepareSimple(p.Name, p.DefaultNamespace, skip)
		if err != nil {
			item.Message = err.Error()
			res.Fail++
			res.Items = append(res.Items, item)
			continue
		}
		st := awsx.CredentialStatusForProfile(r.State.AWSProfile)
		if !st.Valid {
			item.Message = st.Hint
			res.Fail++
			res.Items = append(res.Items, item)
			continue
		}
		if err := kube.ClusterReachable(r.State.Kubeconfig, r.State.KubeContext); err != nil {
			item.Message = err.Error()
			res.Fail++
			res.Items = append(res.Items, item)
			continue
		}
		item.OK = true
		item.Message = fmt.Sprintf("%s / %s", p.Cluster, r.State.KubeContext)
		res.OK++
		res.Items = append(res.Items, item)
	}
	return res
}

func switchNamespace(namespace string) (*session.State, error) {
	s, err := session.Load()
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("no active project")
	}
	if err := kube.UseNamespace(s.Kubeconfig, s.KubeContext, namespace); err != nil {
		return nil, err
	}
	s.Namespace = namespace
	if err := session.Save(s); err != nil {
		return nil, err
	}
	_ = session.RecordRecent(s)
	return s, nil
}

func tailAudit(maxLines int) ([]string, error) {
	path, err := config.AuditPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines, nil
	}
	return lines[len(lines)-maxLines:], nil
}

func findResources(query string) (string, error) {
	s, err := session.Load()
	if err != nil {
		return "", err
	}
	if s == nil {
		return "", fmt.Errorf("no active project")
	}
	return kube.FindResourcesSimple(s.Kubeconfig, s.KubeContext, s.Namespace, query)
}
