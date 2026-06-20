package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
)

// runtime describes an installable language runtime and how to detect it.
type runtime struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	versionCmd  []string
	debianPkgs  []string
	rhelPkgs    []string
}

var runtimes = []runtime{
	{
		Key: "node", Name: "Node.js", Description: "JavaScript-Laufzeit (inkl. npm)",
		versionCmd: []string{"node", "--version"},
		debianPkgs: []string{"nodejs", "npm"}, rhelPkgs: []string{"nodejs", "npm"},
	},
	{
		Key: "python", Name: "Python 3", Description: "Python-Interpreter inkl. pip & venv",
		versionCmd: []string{"python3", "--version"},
		debianPkgs: []string{"python3", "python3-pip", "python3-venv"}, rhelPkgs: []string{"python3", "python3-pip"},
	},
	{
		Key: "java", Name: "Java (OpenJDK)", Description: "Java Development Kit",
		versionCmd: []string{"java", "-version"},
		debianPkgs: []string{"default-jdk"}, rhelPkgs: []string{"java-17-openjdk-devel"},
	},
	{
		Key: "go", Name: "Go", Description: "Go-Compiler & Toolchain",
		versionCmd: []string{"go", "version"},
		debianPkgs: []string{"golang-go"}, rhelPkgs: []string{"golang"},
	},
}

func (r runtime) pkgsFor(family osinfo.Family) []string {
	if family == osinfo.FamilyDebian {
		return r.debianPkgs
	}
	return r.rhelPkgs
}

func findRuntime(key string) (runtime, bool) {
	for _, r := range runtimes {
		if r.Key == key {
			return r, true
		}
	}
	return runtime{}, false
}

type runtimeStatus struct {
	runtime
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

func (s *Server) handleRuntimeList(c *gin.Context) {
	ctx := c.Request.Context()
	out := make([]runtimeStatus, 0, len(runtimes))
	for _, rt := range runtimes {
		st := runtimeStatus{runtime: rt}
		// Most tools print the version to stdout; java prints to stderr.
		res, err := s.runner.Run(ctx, rt.versionCmd[0], rt.versionCmd[1:]...)
		if err == nil {
			st.Installed = true
			line := res.Stdout
			if strings.TrimSpace(line) == "" {
				line = res.Stderr
			}
			st.Version = strings.TrimSpace(strings.SplitN(line, "\n", 2)[0])
		}
		out = append(out, st)
	}
	c.JSON(http.StatusOK, gin.H{"family": s.os.Family, "runtimes": out})
}

func (s *Server) handleRuntimeInstall(c *gin.Context) {
	var req struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "key required")
		return
	}
	rt, ok := findRuntime(req.Key)
	if !ok {
		badRequest(c, "unknown runtime")
		return
	}
	ctx := c.Request.Context()
	var outputs []string
	for _, pkg := range rt.pkgsFor(s.os.Family) {
		out, err := s.pkg.Install(ctx, pkg)
		outputs = append(outputs, out)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "package": pkg, "output": out})
			return
		}
	}
	s.audit(c, "runtime_install", rt.Key)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "output": strings.Join(outputs, "\n")})
}
