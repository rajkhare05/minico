package main

import (
	"fmt"
	"os"
	"os/exec"

	"syscall"
	"golang.org/x/sys/unix"
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

	cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags: unix.CLONE_NEWUTS |
			unix.CLONE_NEWPID |
			unix.CLONE_NEWNS |
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

	must(cmd.Run())
}

func child() {
	fmt.Println("=> Inside container!")

	// clear previous environment
	os.Clearenv()

	// set new environment
	os.Setenv("PATH", "/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	os.Setenv("USER", "root")
	os.Setenv("HOME", "/root")
	os.Setenv("SHELL", "/bin/sh")
	os.Setenv("LANG", "C.UTF-8")

	// Make mount namespace private so it doesn't affect host
	must(unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""))

	// Chroot to minimal filesystem
	rootfs := "/tmp/rootfs"
	must(unix.Chroot(rootfs))
	must(unix.Chdir("/"))

	// Set hostname (UTS namespace)
	// must(unix.Sethostname([]byte("mini-container")))

	// Mount proc, sys, dev inside chroot
	must(unix.Mount("proc", "/proc", "proc", 0, ""))
	must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""))
	must(unix.Mount("tmpfs", "/dev", "tmpfs", 0, ""))

	// Run specified command inside container
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(cmd.Run())

	// Unmount FS
	must(unix.Unmount("/dev", 0))
	must(unix.Unmount("/sys", 0))
	must(unix.Unmount("/proc", 0))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

