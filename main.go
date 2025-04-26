package main

import (
	"fmt"
	"os"
	"os/exec"

	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mini-runtime run <command> [args...]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("Unknown command: " + os.Args[1])
	}
}

func run() {
	fmt.Println("=> Running:", os.Args[2:], "in a new namespace")

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWUSER,

		// UID/GID mapping
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		GidMappingsEnableSetgroups: false,
	}

	must(cmd.Run())
}

func child() {
	fmt.Printf("=> Inside container! PID: %d\n", os.Getpid())

	// Make mount namespace private so it doesn't affect host
	must(syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""))

	// Chroot to minimal filesystem
	rootfs := "/tmp/rootfs"
	must(syscall.Chroot(rootfs))
	must(syscall.Chdir("/"))

	// Set hostname (UTS namespace)
	must(syscall.Sethostname([]byte("mini-container")))

	// Mount proc, sys, dev inside chroot
	must(os.MkdirAll("/proc", 0755))
	must(syscall.Mount("proc", "/proc", "proc", 0, ""))

	must(os.MkdirAll("/sys", 0755))
	must(syscall.Mount("sysfs", "/sys", "sysfs", 0, ""))

	must(os.MkdirAll("/dev", 0755))
	must(syscall.Mount("tmpfs", "/dev", "tmpfs", 0, ""))

	// Run specified command inside container
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(cmd.Run())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

