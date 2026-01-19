package docker

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/saylorsolutions/modmake"
)

type Runner struct {
	runDetached     bool
	runInteractive  bool
	allocateTTY     bool
	attachStdin     bool
	attachStdout    bool
	attachStderr    bool
	runPrivileged   bool
	pullBefore      bool
	pullQuiet       bool
	removeAfterExit bool
	winPaths        bool
	command         []string
	imageName       string
	volumeMapping   map[modmake.PathString]modmake.PathString
	env             map[string]string
	envFile         modmake.PathString
	ports           map[int]int
	hostMapping     map[string]net.IP
	capAdd          []string
	capDrop         []string
	numCpus         float64
	cpuShares       int
	healthCheck     *HealthCheck
	hostName        string
	containerIP     net.IP
	containerIPv6   net.IP
	labels          []string
	memoryLimit     int
	memReserve      int
	containerName   string
	network         string
	stopTimeout     time.Duration
	setUser         string
	workingDir      modmake.PathString
	restartPolicy   string
}

func Run(imageName string) *Runner {
	imageName = strings.TrimSpace(imageName)
	if len(imageName) == 0 {
		panic("empty image name")
	}
	return &Runner{
		imageName:     imageName,
		volumeMapping: map[modmake.PathString]modmake.PathString{},
		env:           map[string]string{},
		ports:         map[int]int{},
	}
}

func (d *Runner) AddHost(host string, ip net.IP) *Runner {
	d.hostMapping[host] = ip
	return d
}

func (d *Runner) AttachStdin() *Runner {
	d.attachStdin = true
	return d
}

func (d *Runner) AttachStdout() *Runner {
	d.attachStdout = true
	return d
}

func (d *Runner) AttachStderr() *Runner {
	d.attachStderr = true
	return d
}

func (d *Runner) AddCapability(cap string) *Runner {
	d.capAdd = append(d.capAdd, cap)
	return d
}

func (d *Runner) DropCapability(cap string) *Runner {
	d.capDrop = append(d.capDrop, cap)
	return d
}

func (d *Runner) CPUs(numCpus float64) *Runner {
	d.numCpus = numCpus
	return d
}

func (d *Runner) CPUShares(shares int) *Runner {
	d.cpuShares = shares
	return d
}

func (d *Runner) UseWindowsContainerPaths() *Runner {
	d.winPaths = true
	return d
}

func (d *Runner) normalizeWinPaths(path modmake.PathString) string {
	if d.winPaths {
		if runtime.GOOS == "windows" {
			return path.String()
		}
		return strings.ReplaceAll(path.String(), "/", "\\")
	}
	return path.ToSlash()
}

// BindMount defines a volume mount mapping a host path to a container path.
func (d *Runner) BindMount(hostPath, mountPath modmake.PathString) *Runner {
	if hostPath.IsBlank() {
		panic("blank host path")
	}
	if mountPath.IsBlank() {
		panic("blank mount path")
	}
	hostPath = hostPath.Abs()
	d.volumeMapping[hostPath] = mountPath
	return d
}

func (d *Runner) BindMountReadOnly(hostPath, mountPath modmake.PathString) *Runner {
	mountPath = modmake.Path(strings.TrimSuffix(d.normalizeWinPaths(mountPath), ":ro") + ":ro")
	return d.BindMount(hostPath, mountPath)
}

func (d *Runner) Detach() *Runner {
	d.runInteractive = false
	d.allocateTTY = false
	d.attachStdin = false
	d.attachStdout = false
	d.attachStderr = false
	d.runDetached = true
	return d
}

func (d *Runner) SetEnv(key, val string) *Runner {
	d.env[key] = val
	return d
}

func (d *Runner) EnvFile(envFile modmake.PathString) *Runner {
	d.envFile = envFile
	return d
}

func (d *Runner) ExposePort(host, container int) *Runner {
	d.ports[host] = container
	return d
}

type HealthCheck struct {
	command       []string
	interval      time.Duration
	numRetries    int
	startInterval time.Duration
	startDelay    time.Duration
	checkTimeout  time.Duration
	disabled      bool
}

func (hc *HealthCheck) setArgs(cmd *modmake.Command) {
	if len(hc.command) == 0 {
		return
	}
	cmd.Arg(fmt.Sprintf("--health-cmd=%s", strings.Join(hc.command, " ")))
	if hc.interval > 0 {
		cmd.Arg("--health-interval", hc.interval.String())
	}
	if hc.numRetries > 0 {
		cmd.Arg("--health-retries", strconv.Itoa(hc.numRetries))
	}
	if hc.startInterval > 0 {
		cmd.Arg("--health-start-interval", hc.startInterval.String())
	}
	if hc.startDelay > 0 {
		cmd.Arg("--health-start-period", hc.startDelay.String())
	}
	if hc.checkTimeout > 0 {
		cmd.Arg("--health-timeout", hc.checkTimeout.String())
	}
}

func disabledHealthCheck() *HealthCheck {
	return &HealthCheck{disabled: true}
}

func NewHealthCheck(cmd string, args ...string) *HealthCheck {
	return &HealthCheck{
		command: append([]string{cmd}, args...),
	}
}

func (hc *HealthCheck) CheckInterval(interval time.Duration) *HealthCheck {
	if interval <= 0 {
		panic("invalid check interval")
	}
	hc.interval = interval
	return hc
}

func (hc *HealthCheck) CheckRetries(num int) *HealthCheck {
	hc.numRetries = num
	return hc
}

func (hc *HealthCheck) StartDelay(delay time.Duration) *HealthCheck {
	hc.startDelay = delay
	return hc
}

func (hc *HealthCheck) StartCheckInterval(interval time.Duration) *HealthCheck {
	hc.startInterval = interval
	return hc
}

func (hc *HealthCheck) CheckTimeout(timeout time.Duration) *HealthCheck {
	hc.checkTimeout = timeout
	return hc
}

func (d *Runner) SetHealthCheck(check *HealthCheck) *Runner {
	d.healthCheck = check
	return d
}

func (d *Runner) NoHealthCheck() *Runner {
	d.healthCheck = disabledHealthCheck()
	return d
}

func (d *Runner) SetHostName(hostName string) *Runner {
	d.hostName = hostName
	return d
}

func (d *Runner) Interactive() *Runner {
	d.runInteractive = true
	return d
}

func (d *Runner) SetIP(ip net.IP) *Runner {
	d.containerIP = ip
	return d
}

func (d *Runner) SetIPv6(ip net.IP) *Runner {
	d.containerIPv6 = ip
	return d
}

func (d *Runner) SetLabel(label string) *Runner {
	d.labels = append(d.labels, label)
	return d
}

func (d *Runner) MemoryLimit(limitBytes int) *Runner {
	d.memoryLimit = limitBytes
	return d
}

func (d *Runner) ReserveMemory(softLimitBytes int) *Runner {
	d.memReserve = softLimitBytes
	return d
}

func (d *Runner) ContainerName(name string) *Runner {
	d.containerName = name
	return d
}

func (d *Runner) JoinNetwork(networkName string) *Runner {
	d.network = networkName
	return d
}

func (d *Runner) Privileged() *Runner {
	d.runPrivileged = true
	return d
}

func (d *Runner) PullBeforeRunning() *Runner {
	d.pullBefore = true
	return d
}

func (d *Runner) PullQuietlyBeforeRunning() *Runner {
	d.pullBefore = true
	d.pullQuiet = true
	return d
}

func (d *Runner) RestartPolicy(policy string) *Runner {
	d.restartPolicy = policy
	return d
}

func (d *Runner) RemoveAfterExit() *Runner {
	d.removeAfterExit = true
	return d
}

func (d *Runner) StopTimeout(timeout time.Duration) *Runner {
	d.stopTimeout = timeout
	return d
}

func (d *Runner) AllocateTTY() *Runner {
	d.allocateTTY = true
	return d
}

func (d *Runner) InteractiveTTY() *Runner {
	d.allocateTTY = true
	d.runInteractive = true
	return d
}

// SetUser sets the running user (and possibly group) with this pattern:
// <name|uid>[:<group|gid>]
func (d *Runner) SetUser(user string) *Runner {
	d.setUser = user
	return d
}

func (d *Runner) SetWorkingDir(containerPath modmake.PathString) *Runner {
	d.workingDir = containerPath
	return d
}

// RunCommand produces a Task that runs the specified command in the container.
func (d *Runner) RunCommand(cmd string, args ...string) modmake.Task {
	return func(ctx context.Context) error {
		d.command = append([]string{cmd}, args...)
		defer func() {
			d.command = nil
		}()
		return d.Run(ctx)
	}
}

// Command resolves the docker CLI, builds the run Command, and returns it for further customization.
func (d *Runner) Command() *modmake.Command {
	exec := modmake.Exec(resolveDockerPath().String(), "run").TrailingArg(d.imageName).LogGroup("docker-run")
	switch {
	case d.runInteractive:
		fallthrough
	case d.attachStdin:
		if !d.runDetached {
			exec.CaptureStdin()
		}
	}
	if len(d.command) > 0 {
		exec.TrailingArg(d.command...)
	}
	for host, ip := range d.hostMapping {
		exec.Arg("--add-host", fmt.Sprintf("%s:%s", host, ip.String()))
	}
	if d.attachStdin {
		exec.Arg("--attach", "STDIN")
	}
	if d.attachStdout {
		exec.Arg("--attach", "STDOUT")
	}
	if d.attachStderr {
		exec.Arg("--attach", "STDERR")
	}
	for _, addedCap := range d.capAdd {
		exec.Arg("--cap-add", addedCap)
	}
	for _, droppedCap := range d.capDrop {
		exec.Arg("--cap-drop", droppedCap)
	}
	if d.numCpus > 0 {
		exec.Arg("--cpus", fmt.Sprintf("%0.2f", d.numCpus))
	}
	if d.cpuShares > 0 {
		exec.Arg("--cpu-shares", fmt.Sprintf("%d", d.cpuShares))
	}
	for host, cont := range d.volumeMapping {
		exec.Arg("-v", fmt.Sprintf("%s:%s", host.String(), cont.ToSlash()))
	}
	if d.runDetached {
		exec.Arg("-d")
	}
	for key, val := range d.env {
		exec.Arg("-e", fmt.Sprintf("%s=%s", key, val))
	}
	if !d.envFile.IsBlank() {
		exec.Arg("--env-file", d.envFile.String())
	}
	for host, cont := range d.ports {
		exec.Arg("-p", fmt.Sprintf("%d:%d", host, cont))
	}
	if d.healthCheck != nil {
		d.healthCheck.setArgs(exec)
	}
	if len(d.hostName) > 0 {
		exec.Arg("--hostname", d.hostName)
	}
	if d.runInteractive {
		exec.Arg("-i")
	}
	if len(d.containerIP) > 0 {
		exec.Arg("--ip", d.containerIP.String())
	}
	if len(d.containerIPv6) > 0 {
		exec.Arg("--ip6", d.containerIPv6.String())
	}
	for _, label := range d.labels {
		exec.Arg("--label", label)
	}
	if d.memoryLimit > 0 {
		exec.Arg("--memory", strconv.Itoa(d.memoryLimit))
	}
	if d.memReserve > 0 {
		exec.Arg("--memory-reservation", strconv.Itoa(d.memReserve))
	}
	if len(d.containerName) > 0 {
		exec.Arg("--name", d.containerName)
	}
	if len(d.network) > 0 {
		exec.Arg("--network", d.network)
	}
	if d.runPrivileged {
		exec.Arg("--privileged")
	}
	if d.pullBefore {
		exec.Arg("--pull")
	}
	if d.pullQuiet {
		exec.Arg("-q")
	}
	if len(d.restartPolicy) > 0 {
		exec.Arg("--restart", d.restartPolicy)
	}
	if d.removeAfterExit {
		exec.Arg("--rm")
	}
	if d.stopTimeout > 0 {
		exec.Arg("--stop-timeout", d.stopTimeout.String())
	}
	if d.allocateTTY {
		exec.Arg("-t")
	}
	if len(d.setUser) > 0 {
		exec.Arg("--user", d.setUser)
	}
	if len(d.workingDir) > 0 {
		exec.Arg("--workdir", d.workingDir.ToSlash())
	}
	return exec
}

func (d *Runner) Task() modmake.Task {
	return d.Run
}

func (d *Runner) Run(ctx context.Context) error {
	return d.Command().Run(ctx)
}

func (d *Runner) String() string {
	return d.Command().String()
}
