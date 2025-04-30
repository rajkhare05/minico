package main

import (
	"fmt"
	"os"
	"os/exec"

	"syscall"
	"golang.org/x/sys/unix"
)

func main() {

	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run()
		os.Exit(1)

	case "child":
		child()
		os.Exit(0)

	default:
		usage()
		fmt.Println("Unknown command: " + os.Args[1])
		os.Exit(1)
	}
}

func run() {

	// setup environment for container
	setupEnv()

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags: unix.CLONE_NEWUTS |
					unix.CLONE_NEWPID |
					unix.CLONE_NEWNS  |
					unix.CLONE_NEWNET |
					unix.CLONE_NEWIPC |
					unix.CLONE_NEWUSER,

		// UID/GID mapping
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		GidMappingsEnableSetgroups: false,
	}


	// must(cmd.Run())
	err := cmd.Start()
	if err != nil {
		os.Exit(1)
	}

	pid := cmd.Process.Pid
	initializeNetworking(pid)
}

func child() {

	// Make mount namespace private so it doesn't affect host
	must(unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""))

	// Chroot to filesystem
	rootfs := "/tmp/rootfs"
	must(unix.Chroot(rootfs))
	must(unix.Chdir("/"))

	// Set hostname (UTS namespace)
	// must(unix.Sethostname([]byte("mini-container")))

	// Mount proc, sys, dev inside chroot
	must(unix.Mount("proc", "/proc", "proc", 0, ""))
	must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""))
	must(unix.Mount("tmpfs", "/dev", "tmpfs", 0, ""))

	addDNSNameserver()

	// Run specified command inside container
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		fmt.Println(err.Error())
	}

	// Unmount
	unMount()
}

func usage() {
	fmt.Println("Usage: minico run <command> [args...]")
}

func setupEnv() {
	// clear old environment
	os.Clearenv()

	// set new environment
	os.Setenv("PATH", "/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	os.Setenv("USER", "root")
	os.Setenv("HOME", "/root")
	os.Setenv("SHELL", "/bin/sh")
	os.Setenv("LANG", "C.UTF-8")
}

// add a network interface in the newly created user namespace
func initializeNetworking(pid int) {
	stringifiedPid := fmt.Sprintf("%d", pid)
	netCmd := exec.Command("slirp4netns", "--configure", "--mtu", "65520", "--disable-host-loopback", stringifiedPid, "veth0")
	if err := netCmd.Run(); err != nil {
		fmt.Println("NetworkError:", err.Error())
	}
}

// add DNS nameserver
func addDNSNameserver() {
	// check if the file exits
	if _, err := os.Stat("/etc/resolv.conf"); err != nil {
		// if not, then
		if os.IsNotExist(err) {
			// create a file
			if _, err = os.Create("/etc/resolv.conf"); err != nil {
				fmt.Println("Error while creating resolv.conf")
				return

			} /* else if os.Chmod("/etc/resolv.conf", 0644) != nil { // change permissions to rw-r--r--
				fmt.Println("Error while changing permissions of resolv.conf")
				return
			} */
		}
	}

	// if the file exits, then write
	if os.WriteFile("/etc/resolv.conf", []byte("nameserver 10.0.2.3"), 0644) != nil {
		fmt.Println("Error while writing resolv.conf")
	}
}

func unMount() {
	must(unix.Unmount("/dev", 0))
	must(unix.Unmount("/sys", 0))
	must(unix.Unmount("/proc", 0))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

