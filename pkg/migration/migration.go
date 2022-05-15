package migration

import (
	"fmt"
	"log"
	"os/exec"
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
}

func Migrate(migs []MigrationCmd, namespace string) {
	for _, m := range migs {
		fmt.Println("Migrating", m.Pod, "wich uses", m.Usage, "GB")
		err := New(m.Pod, namespace).Migrate()
		if err != nil {
			log.Printf("Migration failed: %v", err)
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
