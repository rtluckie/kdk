// Copyright © 2018 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kdk

import (
	"bufio"
	"fmt"
	"github.com/cisco-sso/kdk/pkg/utils"
	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

func Provision(cfg KdkEnvConfig) error {
	log.Info("Starting KDK user provisioning. This may take a moment.  Hang tight...")
	var userProvisionScript, userProvisionScriptPath, userProvisionScriptUrl, userProvisionCommand string

	userProvisionScript = cfg.ConfigFile.AppConfig.UserProvisionScript
	userProvisionCommand = cfg.ConfigFile.AppConfig.UserProvisionCommand

	if utils.ValidUrl(cfg.ConfigFile.AppConfig.UserProvisionScript) {
		log.Info("Using provision script from URL...")
		userProvisionScriptUrl = userProvisionScript
		userProvisionScriptPath = "/tmp/custom_user_provision"
		curlCmd := "curl -Lo %[1]s %[2]s && chmod 0711 %[1]s"
		resp, err := cfg.DockerClient.ContainerExecCreate(cfg.Ctx, cfg.ConfigFile.AppConfig.Name, types.ExecConfig{
			User:         "root",
			Privileged:   true,
			Tty:          false,
			AttachStderr: true,
			AttachStdout: true,
			Cmd:          []string{"bash", "-c", fmt.Sprintf(curlCmd, userProvisionScriptPath, userProvisionScriptUrl)},
		})
		if err != nil {
			log.WithField("error", err).Fatal("Failed to create provision script exec")
			return err
		}
		_, err = cfg.DockerClient.ContainerExecAttach(cfg.Ctx, resp.ID, types.ExecStartCheck{})
		if err != nil {
			log.WithField("error", err).Fatal("Failed to provision using custom script")
			return err
		}
	} else {
		userProvisionScriptPath = userProvisionScript
	}

	createResp, err := cfg.DockerClient.ContainerExecCreate(cfg.Ctx, cfg.ConfigFile.AppConfig.Name, types.ExecConfig{
		User:         "root",
		Privileged:   true,
		Tty:          false,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{userProvisionScriptPath, userProvisionCommand},
	})

	if err != nil {
		log.WithField("error", err).Fatal("Failed to provision")
		return err
	}
	attachResp, err := cfg.DockerClient.ContainerExecAttach(cfg.Ctx, createResp.ID, types.ExecStartCheck{})
	if err != nil {
		log.Fatal(err)
	}
	defer attachResp.Close()
	scanner := bufio.NewScanner(attachResp.Reader)
	for scanner.Scan() {
		log.Debug(fmt.Sprintf("  ⮀ PROVISION: %s", scanner.Text()))
	}
	return nil
}
