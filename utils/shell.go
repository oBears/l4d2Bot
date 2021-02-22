package utils

import (
	"fmt"
	"os/exec"
)

func Command(cmd string) error {
	c := exec.Command("bash", "-c", cmd)
	output, err := c.CombinedOutput()
	fmt.Println(string(output))
	return err
}
