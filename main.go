package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

type Config struct {
	SSHKey  string            `yaml:"ssh_key"`
	Servers map[string]Server `yaml:"servers"`
}

type Server struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Platform string `yaml:"platform"` // "windows" or "linux"
}

type ServerStatus struct {
	Name      string    `json:"name"`
	Online    bool      `json:"online"`
	CPU       string    `json:"cpu"`
	Memory    string    `json:"memory"`
	Disk      string    `json:"disk"`
	GPU       string    `json:"gpu"`
	Uptime    string    `json:"uptime"`
	Tasks     []Task    `json:"tasks"`
	Ports     []Port    `json:"ports,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}

type Port struct {
	Port    string `json:"port"`
	Proto   string `json:"proto"`
	Service string `json:"service"`
	PID     string `json:"pid"`
}

type Task struct {
	PID     string `json:"pid"`
	Name    string `json:"name"`
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Started string `json:"started"`
}

var config Config
var statusCache = make(map[string]*ServerStatus)

func main() {
	// Load config
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal("Failed to read config.yaml:", err)
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatal("Failed to parse config.yaml:", err)
	}

	// Start background updater
	go updateLoop()

	// Web server
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", handleIndex)
	r.GET("/api/status", handleStatus)
	r.POST("/api/report", handleAgentReport)
	r.GET("/api/ports/:server", handlePorts)
	r.GET("/api/logs/:server/:task", handleLogs)
	r.POST("/api/task/:server/start", handleTaskStart)
	r.POST("/api/task/:server/stop", handleTaskStop)

	log.Println("Server starting on :8090")
	r.Run(":8090")
}

func updateLoop() {
	for {
		for name := range config.Servers {
			go updateServerStatus(name)
		}
		time.Sleep(10 * time.Second)
	}
}

func updateServerStatus(name string) {
	server := config.Servers[name]
	status := &ServerStatus{
		Name:      name,
		UpdatedAt: time.Now(),
	}

	client, err := connectSSH(server)
	if err != nil {
		status.Online = false
		status.Error = err.Error()
		statusCache[name] = status
		return
	}
	defer client.Close()

	status.Online = true

	// Query system info
	if server.Platform == "windows" {
		status.CPU = runCommand(client, `powershell -Command "Get-Counter '\Processor(_Total)\% Processor Time' | Select-Object -ExpandProperty CounterSamples | Select-Object -ExpandProperty CookedValue | ForEach-Object { [math]::Round($_, 1) }"`)
		status.Memory = runCommand(client, `powershell -Command "$os = Get-CimInstance Win32_OperatingSystem; [math]::Round(($os.TotalVisibleMemorySize - $os.FreePhysicalMemory) / $os.TotalVisibleMemorySize * 100, 1)"`)
		status.Disk = runCommand(client, `powershell -Command "$d = Get-PSDrive C; [math]::Round($d.Used / ($d.Used + $d.Free) * 100, 1)"`)
		status.GPU = runCommand(client, `nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits`)
		status.Uptime = runCommand(client, `powershell -Command "(Get-Date) - (Get-CimInstance Win32_OperatingSystem).LastBootUpTime | Select-Object -ExpandProperty TotalHours | ForEach-Object { [math]::Round($_, 1) }"`)

		// Get running tasks
		_ = runCommand(client, `powershell -Command "Get-Process python,pythonw -ErrorAction SilentlyContinue | Select-Object Id,ProcessName,CPU,WS,StartTime | ConvertTo-Json -Compress"`)
		// Parse JSON and populate status.Tasks (TODO: implement JSON parsing)

	} else {
		// Linux commands
		status.CPU = runCommand(client, `top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1`)
		status.Memory = runCommand(client, `free | grep Mem | awk '{printf "%.1f", $3/$2 * 100}'`)
		status.Disk = runCommand(client, `df -h / | tail -1 | awk '{print $5}' | tr -d '%'`)
		status.GPU = runCommand(client, `nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>/dev/null || echo "N/A"`)
		status.Uptime = runCommand(client, `uptime -p | cut -d' ' -f2-`)
	}

	statusCache[name] = status
}

func connectSSH(server Server) (*ssh.Client, error) {
	key, err := os.ReadFile(config.SSHKey)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: server.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	return ssh.Dial("tcp", addr, sshConfig)
}

func runCommand(client *ssh.Client, cmd string) string {
	session, err := client.NewSession()
	if err != nil {
		return "error"
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "error"
	}
	return strings.TrimSpace(string(output))
}

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func handleStatus(c *gin.Context) {
	var statuses []ServerStatus
	for _, status := range statusCache {
		statuses = append(statuses, *status)
	}
	c.JSON(http.StatusOK, statuses)
}

func handleAgentReport(c *gin.Context) {
	var report ServerStatus
	if err := c.BindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report.UpdatedAt = time.Now()
	statusCache[report.Name] = &report
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handlePorts(c *gin.Context) {
	serverName := c.Param("server")
	server, ok := config.Servers[serverName]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	client, err := connectSSH(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer client.Close()

	var cmd string
	if server.Platform == "windows" {
		// Use netstat (faster than PowerShell)
		cmd = `netstat -ano | findstr LISTENING`
	} else {
		// Linux/Mac
		cmd = `ss -tulnp 2>/dev/null || netstat -tuln`
	}

	output := runCommand(client, cmd)
	ports := parsePorts(output, server.Platform)
	c.JSON(http.StatusOK, gin.H{"ports": ports})
}

func parsePorts(output string, platform string) []Port {
	var ports []Port
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" || strings.Contains(line, "State") {
			continue
		}

		var port Port
		if platform == "windows" {
			// Windows netstat format: TCP  0.0.0.0:8080  0.0.0.0:0  LISTENING  1234
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				port.Proto = fields[0]
				port.Port = strings.Split(fields[1], ":")[len(strings.Split(fields[1], ":"))-1]
				port.PID = fields[len(fields)-1]
				port.Service = ""
			}
		} else {
			// Linux ss format: tcp   LISTEN 0  128  0.0.0.0:22  0.0.0.0:*  users:(("sshd",pid=123,fd=3))
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				port.Proto = strings.ToUpper(fields[0])
				addr := fields[4]
				if strings.Contains(addr, ":") {
					parts := strings.Split(addr, ":")
					port.Port = parts[len(parts)-1]
				}
				// Extract PID from users:(("name",pid=123,fd=3))
				if len(fields) >= 6 && strings.Contains(fields[5], "pid=") {
					pidPart := strings.Split(fields[5], "pid=")
					if len(pidPart) > 1 {
						port.PID = strings.Split(pidPart[1], ",")[0]
					}
				}
			}
		}

		if port.Port != "" && port.Port != "0" && port.Port != "*" {
			ports = append(ports, port)
		}
	}

	return ports
}

func handleLogs(c *gin.Context) {
	serverName := c.Param("server")
	taskName := c.Param("task")

	server, ok := config.Servers[serverName]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	client, err := connectSSH(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer client.Close()

	var cmd string
	if server.Platform == "windows" {
		cmd = fmt.Sprintf(`powershell -Command "Get-Content C:\jobs\%s.log -Tail 100"`, taskName)
	} else {
		cmd = fmt.Sprintf(`tail -100 /var/log/%s.log`, taskName)
	}

	logs := runCommand(client, cmd)
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func handleTaskStart(c *gin.Context) {
	serverName := c.Param("server")
	var req struct {
		Name string `json:"name"`
		Dir  string `json:"dir"`
		Cmd  string `json:"cmd"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, ok := config.Servers[serverName]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	client, err := connectSSH(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer client.Close()

	var cmd string
	if server.Platform == "windows" {
		cmd = fmt.Sprintf(`powershell -NoProfile -ExecutionPolicy Bypass -File C:\jobs\bg.ps1 -Name %s -Dir "%s" -Cmd "%s"`,
			req.Name, req.Dir, req.Cmd)
	}

	output := runCommand(client, cmd)
	c.JSON(http.StatusOK, gin.H{"output": output})
}

func handleTaskStop(c *gin.Context) {
	serverName := c.Param("server")
	var req struct {
		PID string `json:"pid"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, ok := config.Servers[serverName]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	client, err := connectSSH(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer client.Close()

	var cmd string
	if server.Platform == "windows" {
		cmd = fmt.Sprintf(`taskkill /PID %s /F /T`, req.PID)
	} else {
		cmd = fmt.Sprintf(`kill -9 %s`, req.PID)
	}

	output := runCommand(client, cmd)
	c.JSON(http.StatusOK, gin.H{"output": output})
}
