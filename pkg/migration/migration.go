package migration

import (
	"fmt"
	"os/exec"

	"github.com/containerd/containerd/log"

	"github.com/elchead/k8s-migration-controller/pkg/clock"
)

const kubeconfig = "/home/adrian/config"

type Migration struct {
	Pod       string
	Namespace string
	ScriptDir string
}

type MigrationCmd struct {
	Pod   string
	Usage float64
	NewNode string
	FinishAt clock.Clock
}

func Migrate(migs []MigrationCmd, namespace string) {
	for _, m := range migs {
		log.L.Infof("Migrating %s which uses %f GB", m.Pod, m.Usage)
		err := New(m.Pod, namespace).Migrate()
		if err != nil {
			log.L.Warnf("Migration failed: %v", err)
		}
	}
}

func New(pod, namespace string) *Migration {
	return &Migration{Pod: pod, Namespace: namespace, ScriptDir: "/home/adrian/job-scheduler"}
}

func (m Migration) Migrate() error {
	cmd := exec.Command("/bin/sh", "./tpod_checkpoint.sh")
	cmd.Env = []string{fmt.Sprintf("WORKER=%s", m.Pod), fmt.Sprintf("NS=%s", m.Namespace), fmt.Sprintf("KUBECONFIG=%s", kubeconfig)}
	cmd.Dir = m.ScriptDir
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	return err
}
