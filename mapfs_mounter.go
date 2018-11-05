package nfsv3driver

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"code.cloudfoundry.org/dockerdriver"
	"code.cloudfoundry.org/dockerdriver/driverhttp"
	"code.cloudfoundry.org/dockerdriver/invoker"
	"code.cloudfoundry.org/goshims/ioutilshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/volumedriver"
	"code.cloudfoundry.org/volumedriver/mountchecker"
)

const MAPFS_DIRECTORY_SUFFIX = "_mapfs"
const MAPFS_MOUNT_TIMEOUT = time.Minute * 5

type mapfsMounter struct {
	invoker           invoker.Invoker
	backgroundInvoker BackgroundInvoker
	v3Mounter         volumedriver.Mounter
	osshim            osshim.Os
	ioutilshim        ioutilshim.Ioutil
	mountChecker      mountchecker.MountChecker
	fstype            string
	defaultOpts       string
	resolver          IdResolver
	config            Config
	mapfsPath         string
}

var legacyNfsSharePattern *regexp.Regexp

func init() {
	legacyNfsSharePattern, _ = regexp.Compile("^nfs://([^/]+)(/.*)$")
}

func NewMapfsMounter(invoker invoker.Invoker, bgInvoker BackgroundInvoker, v3Mounter volumedriver.Mounter, osshim osshim.Os, ioutilshim ioutilshim.Ioutil, mountChecker mountchecker.MountChecker, fstype, defaultOpts string, resolver IdResolver, config *Config, mapfsPath string) volumedriver.Mounter {
	return &mapfsMounter{invoker, bgInvoker, v3Mounter, osshim, ioutilshim, mountChecker, fstype, defaultOpts, resolver, *config, mapfsPath}
}

func (m *mapfsMounter) Mount(env dockerdriver.Env, remote string, target string, opts map[string]interface{}) error {
	logger := env.Logger().Session("mount")
	logger.Info("mount-start")
	defer logger.Info("mount-end")

	if _, ok := opts["experimental"]; !ok {
		return m.v3Mounter.Mount(env, remote, target, opts)
	}

	// TODO--refactor the config object so that we don't have to make a local copy just to keep
	// TODO--it from leaking information between mounts.
	tempConfig := m.config.Copy()

	if err := tempConfig.SetEntries(remote, opts, []string{
		"source", "mount", "readonly", "username", "password", "experimental", "version",
	}); err != nil {
		logger.Debug("error-parse-entries", lager.Data{
			"given_source":  remote,
			"given_target":  target,
			"given_options": opts,
			"config_source": tempConfig.source,
			"config_mounts": tempConfig.mount,
			"config_sloppy": tempConfig.sloppyMount,
		})
		return dockerdriver.SafeError{SafeDescription: err.Error()}
	}

	if username, ok := opts["username"]; ok {
		if m.resolver == nil {
			return dockerdriver.SafeError{SafeDescription: "LDAP username is specified but LDAP is not configured"}
		}
		password, ok := opts["password"]
		if !ok {
			return dockerdriver.SafeError{SafeDescription: "LDAP username is specified but LDAP password is missing"}
		}

		uid, gid, err := m.resolver.Resolve(env, username.(string), password.(string))
		if err != nil {
			// this error is not wrapped in SafeError since it might contain sensitive information
			return err
		}

		opts["uid"] = uid
		opts["gid"] = gid
		tempConfig.source.Allowed = append(tempConfig.source.Allowed, "uid", "gid")
		if err := tempConfig.SetEntries(remote, opts, []string{
			"source", "mount", "readonly", "username", "password", "experimental",
		}); err != nil {
			return dockerdriver.SafeError{SafeDescription: err.Error()}
		}
	}

	_, uidok := opts["uid"]
	_, gidok := opts["gid"]
	if uidok && !gidok {
		return dockerdriver.SafeError{SafeDescription: "required 'gid' option is missing"}
	}

	// check for legacy URL formatted mounts and rewrite to standard nfs format as necessary
	match := legacyNfsSharePattern.FindStringSubmatch(remote)
	if len(match) > 2 {
		remote = match[1] + ":" + match[2]
	}

	target = strings.TrimSuffix(target, "/")

	intermediateMount := target + MAPFS_DIRECTORY_SUFFIX
	orig := syscall.Umask(000)
	defer syscall.Umask(orig)
	err := m.osshim.MkdirAll(intermediateMount, os.ModePerm)
	if err != nil {
		logger.Error("mkdir-intermediate-failed", err)
		return dockerdriver.SafeError{SafeDescription: err.Error()}
	}

	mountOptions := m.defaultOpts
	if _, ok := opts["readonly"]; ok {
		mountOptions = strings.Replace(mountOptions, ",actimeo=0", "", -1)
	}

	if _, ok := opts["version"]; ok {
		mountOptions = mountOptions + ",vers=" + opts["version"].(string)
	} else {
		mountOptions = mountOptions + ",vers=3"
	}

	t := intermediateMount
	if !uidok {
		t = target
	}

	_, err = m.invoker.Invoke(env, "mount", []string{"-t", m.fstype, "-o", mountOptions, remote, t})
	if err != nil {
		logger.Error("invoke-mount-failed", err)
		m.osshim.RemoveAll(intermediateMount)
		return dockerdriver.SafeError{SafeDescription: err.Error()}
	}

	if uidok {
		args := tempConfig.MapfsOptions()
		args = append(args, target, intermediateMount)
		err = m.backgroundInvoker.Invoke(env, m.mapfsPath, args, "Mounted!", MAPFS_MOUNT_TIMEOUT)
		if err != nil {
			logger.Error("background-invoke-mount-failed", err)
			m.invoker.Invoke(env, "umount", []string{intermediateMount})
			m.osshim.Remove(intermediateMount)
			return dockerdriver.SafeError{SafeDescription: err.Error()}
		}
	}

	return nil
}

func (m *mapfsMounter) Unmount(env dockerdriver.Env, target string) error {
	logger := env.Logger().Session("unmount")
	logger.Info("unmount-start")
	defer logger.Info("unmount-end")

	target = strings.TrimSuffix(target, "/")
	intermediateMount := target + MAPFS_DIRECTORY_SUFFIX

	exists, e := m.mountChecker.Exists(intermediateMount)
	if e != nil {
		return dockerdriver.SafeError{SafeDescription: e.Error()}
	}

	if !exists {
		return m.v3Mounter.Unmount(env, target)
	}

	if _, e := m.invoker.Invoke(env, "umount", []string{"-l", target}); e != nil {
		return dockerdriver.SafeError{SafeDescription: e.Error()}
	}

	if _, e := m.invoker.Invoke(env, "umount", []string{"-l", intermediateMount}); e != nil {
		// this error may be benign since mounts without uid don't actually use this directory
		logger.Error("warning-umount-intermediate-failed", e)
	}

	if e := m.osshim.Remove(intermediateMount); e != nil {
		return dockerdriver.SafeError{SafeDescription: e.Error()}
	}

	return nil
}

func (m *mapfsMounter) Check(env dockerdriver.Env, name, mountPoint string) bool {
	logger := env.Logger().Session("check")
	logger.Info("check-start")
	defer logger.Info("check-end")

	ctx, _ := context.WithDeadline(context.TODO(), time.Now().Add(time.Second*5))
	env = driverhttp.EnvWithContext(ctx, env)
	_, err := m.invoker.Invoke(env, "mountpoint", []string{"-q", mountPoint})

	if err != nil {
		logger.Info(fmt.Sprintf("unable to verify volume %s (%s)", name, err.Error()))
		return false
	}
	return true
}

func (m *mapfsMounter) Purge(env dockerdriver.Env, path string) {
	logger := env.Logger().Session("purge")
	logger.Info("purge-start")
	defer logger.Info("purge-end")

	output, err := m.invoker.Invoke(env, "pkill", []string{"mapfs"})
	logger.Info("pkill", lager.Data{"output": output, "err": err})

	for i := 0; i < 30 && err == nil; i++ {
		logger.Info("waiting-for-kill")
		time.Sleep(PurgeTimeToSleep)
		output, err = m.invoker.Invoke(env, "pgrep", []string{"mapfs"})
		logger.Info("pgrep", lager.Data{"output": output, "err": err})
	}

	mounts, err := m.mountChecker.List("^" + path + ".*" + MAPFS_DIRECTORY_SUFFIX + "$")
	if err != nil {
		logger.Error("check-proc-mounts-failed", err, lager.Data{"path": path})
		return
	}

	logger.Info("mount-directory-list", lager.Data{"mounts": mounts})

	for _, mountDir := range mounts {
		realMountpoint := strings.TrimSuffix(mountDir, MAPFS_DIRECTORY_SUFFIX)

		_, err = m.invoker.Invoke(env, "umount", []string{"-l", "-f", realMountpoint})
		if err != nil {
			logger.Error("warning-umount-intermediate-failed", err)
		}

		logger.Info("unmount-successful", lager.Data{"path": realMountpoint})

		if err := m.osshim.Remove(realMountpoint); err != nil {
			logger.Error("purge-cannot-remove-directory", err, lager.Data{"name": realMountpoint, "path": path})
		}

		logger.Info("remove-directory-successful", lager.Data{"path": realMountpoint})

		_, err = m.invoker.Invoke(env, "umount", []string{"-l", "-f", mountDir})
		if err != nil {
			logger.Error("warning-umount-mapfs-failed", err)
		}

		logger.Info("unmount-successful", lager.Data{"path": mountDir})

		if err := m.osshim.Remove(mountDir); err != nil {
			logger.Error("purge-cannot-remove-directory", err, lager.Data{"name": mountDir, "path": path})
		}

		logger.Info("remove-directory-successful", lager.Data{"path": mountDir})
	}

	// TODO -- when we remove the legacy mounter, replace this with something that just deletes all the remaining
	// TODO -- directories
	m.v3Mounter.Purge(env, path)
}
