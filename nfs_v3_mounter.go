package nfsv3driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"strings"

	"code.cloudfoundry.org/goshims/ioutilshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/nfsdriver"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
	"code.cloudfoundry.org/voldriver/invoker"
	"path/filepath"
)

type nfsV3Mounter struct {
	invoker  invoker.Invoker
	osutil   osshim.Os
	ioutil   ioutilshim.Ioutil
	config   Config
	resolver IdResolver
}

var PurgeTimeToSleep = time.Millisecond * 100

func NewNfsV3Mounter(invoker invoker.Invoker, osutil osshim.Os, ioutil ioutilshim.Ioutil, config *Config, resolver IdResolver) nfsdriver.Mounter {
	return &nfsV3Mounter{invoker: invoker, osutil: osutil, ioutil: ioutil, config: *config, resolver: resolver}
}

func (m *nfsV3Mounter) Mount(env voldriver.Env, source string, target string, opts map[string]interface{}) error {
	logger := env.Logger().Session("fuse-nfs-mount")
	logger.Info("start")
	defer logger.Info("end")

	// TODO--refactor the config object so that we don't have to make a local copy just to keep
	// TODO--it from leaking information between mounts.
	tempConfig := m.config.Copy()

	if err := tempConfig.SetEntries(source, opts, []string{
		"source", "mount", "kerberosPrincipal", "kerberosKeytab", "readonly", "username", "password",
	}); err != nil {
		logger.Debug("error-parse-entries", lager.Data{
			"given_source":  source,
			"given_target":  target,
			"given_options": opts,
			"config_source": tempConfig.source,
			"config_mounts": tempConfig.mount,
			"config_sloppy": tempConfig.sloppyMount,
		})
		return err
	}

	if username, ok := opts["username"]; ok {
		if m.resolver == nil {
			return errors.New("LDAP username is specified but LDAP is not configured")
		}
		password, ok := opts["password"]
		if !ok {
			return errors.New("LDAP username is specified but LDAP password is missing")
		}

		tempConfig.source.Allowed = append(tempConfig.source.Allowed, "uid", "gid")

		uid, gid, err := m.resolver.Resolve(env, username.(string), password.(string))
		if err != nil {
			return err
		}

		opts["uid"] = uid
		opts["gid"] = gid
		err = tempConfig.SetEntries(source, opts, []string{
			"source", "mount", "kerberosPrincipal", "kerberosKeytab", "readonly", "username", "password",
		})
		if err != nil {
			return err
		}
	}

	mountOptions := append([]string{
		"-a",
		"-n", tempConfig.Share(source),
		"-m", target,
	}, tempConfig.Mount()...)

	if _, ok := opts["readonly"]; ok {
		mountOptions = append(mountOptions, "-O")
	}

	logger.Debug("parse-mount", lager.Data{
		"given_source":  source,
		"given_target":  target,
		"given_options": opts,
		"config_source": tempConfig.source,
		"config_mounts": tempConfig.mount,
		"config_sloppy": tempConfig.sloppyMount,
		"mountOptions":  mountOptions,
	})

	logger.Debug("exec-mount", lager.Data{"params": strings.Join(mountOptions, ",")})
	_, err := m.invoker.Invoke(env, "fuse-nfs", mountOptions)
	if err != nil {
		logger.Error("fuse-nfs-invocation-failed", err)
		m.invoker.Invoke(env, "fusermount", []string{"-u", target})
	}
	return err
}

func (m *nfsV3Mounter) Unmount(env voldriver.Env, target string) error {
	_, err := m.invoker.Invoke(env, "fusermount", []string{"-u", target})
	return err
}

func (m *nfsV3Mounter) Check(env voldriver.Env, name, mountPoint string) bool {
	ctx, _ := context.WithDeadline(context.TODO(), time.Now().Add(time.Second*5))
	env = driverhttp.EnvWithContext(ctx, env)
	_, err := m.invoker.Invoke(env, "mountpoint", []string{"-q", mountPoint})

	if err != nil {
		// Note: Created volumes (with no mounts) will be removed
		//       since VolumeInfo.Mountpoint will be an empty string
		env.Logger().Info(fmt.Sprintf("unable to verify volume %s (%s)", name, err.Error()))
		return false
	}
	return true
}

func (m *nfsV3Mounter) Purge(env voldriver.Env, path string) {
	logger := env.Logger().Session("purge")
	logger.Info("start")
	defer logger.Info("end")

	output, err := m.invoker.Invoke(env, "pkill", []string{"fuse-nfs"})
	logger.Info("pkill", lager.Data{"output": output, "err": err})

	for i := 0; i < 30 && err == nil; i++ {
		logger.Info("waiting-for-kill")
		time.Sleep(PurgeTimeToSleep)
		output, err = m.invoker.Invoke(env, "pgrep", []string{"fuse-nfs"})
		logger.Info("pgrep", lager.Data{"output": output, "err": err})
	}

	if err == nil {
		logger.Info("warning-fuse-nfs-not-terminated")
	}

	fileInfos, err := m.ioutil.ReadDir(path)
	if err != nil {
		env.Logger().Error("purge-readdir-failed", err, lager.Data{"path": path})
		return
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			if err := m.osutil.RemoveAll(filepath.Join(path, fileInfo.Name())); err != nil {
				env.Logger().Error("purge-cannot-remove-directory", err, lager.Data{"name": fileInfo.Name(), "path": path})
			}
		}
	}
}
