package docker

import (
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Run struct {
	inst            *Inst
	runDetatched    bool
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

func (d *Inst) Run(imageName string) *Run {
	imageName = strings.TrimSpace(imageName)
	if len(imageName) == 0 {
		panic("empty image name")
	}
	return &Run{
		inst:          d,
		imageName:     imageName,
		volumeMapping: map[modmake.PathString]modmake.PathString{},
		env:           map[string]string{},
		ports:         map[int]int{},
	}
}

func (d *Run) AddHost(host string, ip net.IP) *Run {
	d.hostMapping[host] = ip
	return d
}

func (d *Run) AttachStdin() *Run {
	d.attachStdin = true
	return d
}

func (d *Run) AttachStdout() *Run {
	d.attachStdout = true
	return d
}

func (d *Run) AttachStderr() *Run {
	d.attachStderr = true
	return d
}

func (d *Run) AddCapability(cap string) *Run {
	d.capAdd = append(d.capAdd, cap)
	return d
}

func (d *Run) DropCapability(cap string) *Run {
	d.capDrop = append(d.capDrop, cap)
	return d
}

func (d *Run) CPUs(numCpus float64) *Run {
	d.numCpus = numCpus
	return d
}

func (d *Run) CPUShares(shares int) *Run {
	d.cpuShares = shares
	return d
}

func (d *Run) UseWindowsContainerPaths() *Run {
	d.winPaths = true
	return d
}

func (d *Run) normalizeWinPaths(path modmake.PathString) string {
	if d.winPaths {
		if runtime.GOOS == "windows" {
			return path.String()
		}
		return strings.ReplaceAll(path.String(), "/", "\\")
	}
	return path.ToSlash()
}

// BindMount defines a volume mount mapping a host path to a container path.
func (d *Run) BindMount(hostPath, mountPath modmake.PathString) *Run {
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

func (d *Run) BindMountReadOnly(hostPath, mountPath modmake.PathString) *Run {
	mountPath = modmake.Path(strings.TrimSuffix(d.normalizeWinPaths(mountPath), ":ro") + ":ro")
	return d.BindMount(hostPath, mountPath)
}

func (d *Run) Detach() *Run {
	d.runInteractive = false
	d.allocateTTY = false
	d.attachStdin = false
	d.attachStdout = false
	d.attachStderr = false
	d.runDetatched = true
	return d
}

func (d *Run) SetEnv(key, val string) *Run {
	d.env[key] = val
	return d
}

func (d *Run) EnvFile(envFile modmake.PathString) *Run {
	d.envFile = envFile
	return d
}

func (d *Run) ExposePort(host, container int) *Run {
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

func (d *Run) SetHealthCheck(check *HealthCheck) *Run {
	d.healthCheck = check
	return d
}

func (d *Run) NoHealthCheck() *Run {
	d.healthCheck = disabledHealthCheck()
	return d
}

func (d *Run) SetHostName(hostName string) *Run {
	d.hostName = hostName
	return d
}

func (d *Run) Interactive() *Run {
	d.runInteractive = true
	return d
}

func (d *Run) SetIP(ip net.IP) *Run {
	d.containerIP = ip
	return d
}

func (d *Run) SetIPv6(ip net.IP) *Run {
	d.containerIPv6 = ip
	return d
}

func (d *Run) SetLabel(label string) *Run {
	d.labels = append(d.labels, label)
	return d
}

func (d *Run) MemoryLimit(limitBytes int) *Run {
	d.memoryLimit = limitBytes
	return d
}

func (d *Run) ReserveMemory(softLimitBytes int) *Run {
	d.memReserve = softLimitBytes
	return d
}

func (d *Run) ContainerName(name string) *Run {
	d.containerName = name
	return d
}

func (d *Run) JoinNetwork(networkName string) *Run {
	d.network = networkName
	return d
}

func (d *Run) Privileged() *Run {
	d.runPrivileged = true
	return d
}

func (d *Run) PullBeforeRunning() *Run {
	d.pullBefore = true
	return d
}

func (d *Run) PullQuietlyBeforeRunning() *Run {
	d.pullBefore = true
	d.pullQuiet = true
	return d
}

func (d *Run) RestartPolicy(policy string) *Run {
	d.restartPolicy = policy
	return d
}

func (d *Run) RemoveAfterExit() *Run {
	d.removeAfterExit = true
	return d
}

func (d *Run) StopTimeout(timeout time.Duration) *Run {
	d.stopTimeout = timeout
	return d
}

func (d *Run) AllocateTTY() *Run {
	d.allocateTTY = true
	return d
}

func (d *Run) InteractiveTTY() *Run {
	d.allocateTTY = true
	d.runInteractive = true
	return d
}

// SetUser sets the running user (and possibly group) with this pattern:
// <name|uid>[:<group|gid>]
func (d *Run) SetUser(user string) *Run {
	d.setUser = user
	return d
}

func (d *Run) SetWorkingDir(containerPath modmake.PathString) *Run {
	d.workingDir = containerPath
	return d
}

// RunCommand produces a Task that runs the specified command in the container.
func (d *Run) RunCommand(cmd string, args ...string) modmake.Task {
	return func(ctx context.Context) error {
		d.command = append([]string{cmd}, args...)
		defer func() {
			d.command = nil
		}()
		return d.Run(ctx)
	}
}

func (d *Run) Command() *modmake.Command {
	exec := modmake.Exec(d.inst.dockerPath.String(), "run").TrailingArg(d.imageName)
	switch {
	case d.runInteractive:
		fallthrough
	case d.attachStdin:
		if !d.runDetatched {
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
	if d.runDetatched {
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

func (d *Run) Task() modmake.Task {
	return d.Run
}

func (d *Run) Run(ctx context.Context) error {
	return d.Command().Run(ctx)
}

func (d *Run) String() string {
	return d.Command().String()
}
